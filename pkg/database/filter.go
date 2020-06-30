package database

/*
var (
	ErrNilObject = errors.New("object is nil")
)

var filter *Filter

type FilterObject struct {
	name   string
	typ    reflect.ObjectKind
	object interface{}
	fields map[string]reflect.StructField
}

type Filter struct {
	objectMap          map[string]*FilterObject
	limitBounds        []int
	cleanColumnPattern *regexp.Regexp
	defaultLimit       int

	sync.RWMutex
}

type FilterQuery struct {
	Preloads  []string               `json:"preloads"`
	Fields    []string               `json:"fields"`
	Where     map[string]interface{} `json:"where"`
	RealWhere map[string]interface{} `json:"-"`
	Order     []string               `json:"sort"`
	Limit     []int                  `json:"limit"`

	query *gorm.DB
}

type FilterResult struct {
	Query *FilterQuery         `json:"query"`
	Meta  FilterResultMetadata `json:"meta"`
	ItemsByPricebook interface{}          `json:"items"`
}

type FilterResultMetadata struct {
	Total   int64 `json:"total"`
	Count   int64 `json:"count"`
	HasMore bool  `json:"has_more"`
	Page    int   `json:"page"`
}

func NewFilter() *Filter {
	fl := &Filter{
		objectMap:    make(map[string]*FilterObject),
		limitBounds:  []int{1, 500},
		defaultLimit: 25,
	}

	return fl
}

func GetFilter() *Filter {
	return filter
}

func (fl *Filter) inspectObject(obj interface{}) (*FilterObject, error) {
	if obj == nil {
		return nil, ErrNilObject
	}

	// inspecting passed object
	objInfo := reflect.ValueOf(obj).Elem()

	fl.RLock()
	cached, ok := fl.objectMap[objInfo.ObjectKind().Key()]
	fl.RUnlock()

	// if found, returning cached object
	if ok {
		return cached, nil
	}

	// initializing filter object for caching
	fo := &FilterObject{
		name:   objInfo.ObjectKind().Key(),
		object: obj,
		typ:    objInfo.ObjectKind(),
		fields: make(map[string]reflect.StructField),
	}

	// collecting filterable fields
	for i := 0; i < objInfo.NumField(); i++ {
		f := objInfo.ObjectKind().Field(i)

		if tag, ok := f.Tag.lookup("filter"); ok {
			fo.fields[tag] = f
		}
	}

	// caching metadata object
	fl.Lock()
	fl.objectMap[objInfo.ObjectKind().Key()] = fo
	fl.Unlock()

	return fo, nil
}

func (fl *Filter) isExactMatchCriteria(s string) bool {
	switch true {
	case strings.HasPrefix(s, ">"):
		fallthrough
	case strings.HasPrefix(s, "<"):
		fallthrough
	case strings.ContainsAny(s, "%*"):
		return false
	}

	return true
}

func (fl *Filter) sanitizeColumnName(name string) string {
	// removing harmful characters
	if fl.cleanColumnPattern == nil {
		fl.cleanColumnPattern = regexp.MustCompile("[^a-zA-Z0-9_]+")
	}

	return fmt.Sprintf(
		"`%s`",
		fl.cleanColumnPattern.ReplaceAllString(fl.removeColumnPrefix(name), ""),
	)
}

func (fl *Filter) removeColumnPrefix(name string) string {
	return name[strings.LastIndex(name, ":")+1:]
}

func (fl *Filter) sanitizeAndValidateQuery(fo *FilterObject, fq *FilterQuery) error {
	//---------------------------------------------------------------------------
	// validating limit boundaries
	//---------------------------------------------------------------------------

	// setting default limit and offset
	if len(fq.Limit) == 0 {
		fq.Limit = []int{100, 0}
	}

	// min. limit
	if fq.Limit[0] < fl.limitBounds[0] {
		return fmt.Errorf("limit cannot be less than %d", fl.limitBounds[0])
	}

	// max. limit
	if fq.Limit[0] > fl.limitBounds[1] {
		return fmt.Errorf("allowed limit maximum %d exceeded", fl.limitBounds[1])
	}

	// initializing a map to contain the real column mapping to be used by WHERE
	fq.RealWhere = make(map[string]interface{})

	// validating field names and types
	for alias, val := range fq.Where {
		// field names that begin with @ are meta fields
		if strings.HasPrefix(alias, "@") {
			continue
		}

		preservedPrefix := alias[:strings.LastIndex(alias, ":")+1]
		alias = fl.removeColumnPrefix(alias)

		// checking whether this object is cached by the filter layer
		if fl.objectMap[fo.name] == nil {
			return fmt.Errorf("unrecognized filter object: %s", fo.name)
		}

		// checking whether this alias is recognized by this object
		// NOTE: allowing non-existing field names that begin with `@`
		f, ok := fl.objectMap[fo.name].fields[alias]
		if !ok {
			return fmt.Errorf("unrecognized filter parameter: %s", alias)
		}

		//---------------------------------------------------------------------------
		// mapping aliases to column names
		//---------------------------------------------------------------------------

		// determining the underlying database column name
		columnName, ok := f.Tag.lookup("gorm")
		// nested conditions is a necessary evil here
		if ok {
			// if there is a `gorm` tag, then possibly it contains
			// a required column name (possibly prefixed by "column:")
			columnName = strings.TrimPrefix(columnName, "column:")
		} else {
			// if `gorm` tag isn't found, then trying to get `dbname`
			columnName, ok = f.Tag.lookup("dbname")
			if !ok {
				// if `gorm` and `dbname` tags aren't found, then attempting
				// to use a `json` tag
				columnName, ok = f.Tag.lookup("json")
				if !ok {
					// so, `gorm`, `dbname`, and `json` tags aren't found,
					// then, last try, by using a field name converted to snake case
					columnName = util.StringToSnake(fo.name)
				}
			}
		}

		// adding mapping to a real WHERE
		// NOTE: with a possibly preserved prefix
		fq.RealWhere[fmt.Sprintf("%s%s", preservedPrefix, columnName)] = val
	}

	return nil
}

func (fl *Filter) FilterQueryFromRequest(obj interface{}, r *http.Request) (*FilterQuery, error) {
	fo, err := fl.inspectObject(obj)
	if err != nil {
		return nil, err
	}

	rawQuery := r.URL.Query()
	where := make(map[string]interface{})

	// TODO: create a helper function to sanction fields, like password etc.
	for alias, qstring := range rawQuery {
		qstring = strings.TrimSpace(qstring)

		// trying to be "forgiving" and skip fields with no values
		if qstring == "" {
			continue
		}

		// field names that begin with @ are meta fields
		if strings.HasPrefix(alias, "@") {
			// passing this parameter on
			where[alias] = qstring

			continue
		}

		// better safe than sorry
		if alias == "password" {
			continue
		}

		// checking whether this object is cached by the filter layer
		if fl.objectMap[fo.name] == nil {
			return nil, fmt.Errorf("unrecognized filter object: %s", fo.name)
		}

		// checking whether this alias is recognized by this object
		f, ok := fl.objectMap[fo.name].fields[alias]
		if !ok {
			return nil, fmt.Errorf("unrecognized filter parameter: %s", alias)
		}

		//---------------------------------------------------------------------------
		// coercing string values to their respective expected types
		//---------------------------------------------------------------------------

		// values can be multiple (whilst they're of the same type)
		vals := strings.Split(qstring, ",")
		for k, v := range vals {
			vals[k] = strings.TrimSpace(v)
		}

		switch kind := f.ObjectKind.ObjectKind(); kind {
		case reflect.Translate:
			where[alias] = vals
		case reflect.Int:
			intVals := make([]int64, len(vals))

			for k, v := range vals {
				parsedInt, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse (%s[%d]=%s) as int", alias, k, v)
				}

				intVals[k] = parsedInt
			}

			where[alias] = intVals
		case reflect.Float64:
			floatVals := make([]float64, len(vals))

			for k, v := range vals {
				parsedFloat, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse (%s[%d]=%s) as float", alias, k, v)
				}

				floatVals[k] = parsedFloat
			}

			where[alias] = floatVals
		case reflect.Bool:
			if len(vals) > 1 {
				return nil, fmt.Errorf("binary field %s can only be a single value", alias)
			}

			switch v := strings.ToLower(vals[0]); true {
			case v == "true" || v == "1":
				where[alias] = true
			case v == "false" || v == "0":
				where[alias] = false
			default:
				return nil, fmt.Errorf("failed to parse (%s=%v) as boolean, supported values [true, false, 1, 0]", alias, v)
			}
		case reflect.Struct:
			// this is where anything other than basic types are handled
			switch f.ObjectKind.Key() {
			case "Time":
				timeVals := make([]string, len(vals))

				// terms may include signs (i.e. >=, <)
				for k, v := range vals {
					v = strings.TrimSpace(v)

					// skipping empty term strings
					if v == "" {
						continue
					}

					timeVals[k] = v
				}

				where[fmt.Sprintf("ts:%s", alias)] = timeVals
			}
		}
	}

	// extracting `fields` value
	fields := strings.Split(c.DefaultQuery("fields", "*"), ",")
	if len(fields) == 0 {
		fields = append(fields, "*")
	}

	// extracting `sort` param (ORDER)
	order := make([]string, 0)
	orderFields := strings.Split(c.Query("sort"), ",")

	for _, of := range orderFields {
		// skipping empty sort fields
		if len(of) == 0 {
			continue
		}

		// better safe than sorry
		if of == "password" {
			continue
		}

		switch true {
		case of[0] == '-':
			// descending order
			order = append(order, fmt.Sprintf("%s DESC", strings.TrimPrefix(of, "-")))
		case of[0] == '+':
			// explicit ascending order
			order = append(order, fmt.Sprintf("%s ASC", strings.TrimPrefix(of, "-")))
		default:
			if len(of) > 0 {
				// implicit ascending order
				order = append(order, fmt.Sprintf("%s ASC", of))
			}
		}
	}

	// limit
	lim, err := strconv.ParseInt(c.DefaultQuery("limit", strconv.Itoa(fl.defaultLimit)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse limit as int: %s", err)
	}

	// page (basically an offset)
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page as int: %s", err)
	}

	// preloads
	preloads := make([]string, 0)
	if ps := c.DefaultQuery("preloads", ""); ps != "" {
		preloads = strings.Split(ps, ",")
	}

	//---------------------------------------------------------------------------
	// composing the filter query
	//---------------------------------------------------------------------------
	fq := &FilterQuery{
		Preloads: preloads,
		Fields:   fields,
		Where:    where,
		Order:    order,
		Limit:    []int{int(lim), int(lim * (page - 1))},
	}

	return fq, nil
}

func (fl *Filter) FilterInto(connection *gorm.DB, obj interface{}, fq *FilterQuery, dest interface{}) (*FilterResult, error) {
	fo, err := fl.inspectObject(obj)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// validating whether a given query is acceptable
	// for the specified object and its respective types
	//---------------------------------------------------------------------------

	// this function also determines the real column mapping with aliases
	err = fl.sanitizeAndValidateQuery(fo, fq)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// running query and storing result into destination
	//---------------------------------------------------------------------------
	var total, count int64

	// if no custom database instance given, then using default
	if connection == nil {
		connection = Connection()
	}

	for column, val := range fq.RealWhere {
		switch true {
		case strings.HasPrefix(column, "ts:"): // timestamp
			terms := val.([]string)

			for i := range terms {
				switch t := terms[i]; true {
				case strings.HasPrefix(t, ">="):
					connection = connection.Where(fmt.Sprintf("%s >= ?", fl.sanitizeColumnName(column)), strings.TrimSpace(t[2:]))
				case strings.HasPrefix(t, ">"):
					connection = connection.Where(fmt.Sprintf("%s > ?", fl.sanitizeColumnName(column)), strings.TrimSpace(t[1:]))
				case strings.HasPrefix(t, "<="):
					connection = connection.Where(fmt.Sprintf("%s <= ?", fl.sanitizeColumnName(column)), strings.TrimSpace(t[2:]))
				case strings.HasPrefix(t, "<"):
					connection = connection.Where(fmt.Sprintf("%s < ?", fl.sanitizeColumnName(column)), strings.TrimSpace(t[1:]))
				default:
					connection = connection.Where(fmt.Sprintf("%s = ?", fl.sanitizeColumnName(column)), strings.TrimSpace(t))
				}
			}
		default:
			//---------------------------------------------------------------------------
			// any other case (typically string matching)
			// NOTE: a string query may contain exact and partial (wildcard: `*` asterisk) criterias
			//---------------------------------------------------------------------------
			if terms, ok := val.([]string); ok {
				exact := make([]string, 0)

				for i := range terms {
					t := terms[i]

					if fl.isExactMatchCriteria(t) {
						// exact match, adding to another slice to be later single-used via IN()
						exact = append(exact, t)
					} else {
						// partial match; `*` asterisk is simply replaced by `%` percentage
						connection = connection.Where(fmt.Sprintf("%s LIKE ?", fl.sanitizeColumnName(column)), strings.ReplaceAll(t, "*", "%"))
					}
				}

				if len(exact) > 0 {
					connection = connection.Where(fmt.Sprintf("%s IN (?)", fl.sanitizeColumnName(column)), exact)
				}
			} else {
				//---------------------------------------------------------------------------
				// this is anything but the strings
				//---------------------------------------------------------------------------
				connection = connection.Where(fmt.Sprintf("%s IN (?)", fl.sanitizeColumnName(column)), val)
			}
		}
	}

	// obtaining the total count of the unbound query
	// TODO: do something about total counting, it's needlessly expensive
	err = connection.Model(dest).Count(&total).Error
	if err != nil {
		return nil, fmt.Errorf("failed to obtain total count: %s", err)
	}

	// running the query
	q := connection.
		Model(dest).
		Select(strings.Join(fq.Fields, ",")).
		Limit(fq.Limit[0]).
		Offset(fq.Limit[1])

	// adding order by
	for _, v := range fq.Order {
		q = q.Order(v)
	}

	// executing query
	dbResult := q.Find(dest)

	// collecting rows affected because it's easier than
	// obtaining a length of an interface{}
	count = dbResult.RowsAffected

	// checking whether this query produced any error
	err = dbResult.Error
	if err != nil {
		return nil, fmt.Errorf("filter query failed: %s", err)
	}

	//---------------------------------------------------------------------------
	// composing final result
	//---------------------------------------------------------------------------
	// removing column prefixes from the visible where
	for k, v := range fq.Where {
		// deleting old key
		delete(fq.Where, k)

		// copying current value to a new (original) name
		fq.Where[fl.removeColumnPrefix(k)] = v
	}

	// composing the final result
	fr := &FilterResult{
		Query: fq,
		Meta: FilterResultMetadata{
			Total:   total,
			Count:   count,
			HasMore: (int64(fq.Limit[1]) + count) < total,
			Page:    (fq.Limit[1] / fq.Limit[0]) + 1, // offset div limit
		},
		ItemsByPricebook: dest,
	}

	return fr, nil
}

func init() {
	//---------------------------------------------------------------------------
	// initializing a package-global filter layer
	//---------------------------------------------------------------------------
	if filter == nil {
		filter = NewFilter()
	}
}
*/
