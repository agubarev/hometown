package guard

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

var instance *cache

func init() {
	instance = &cache{
		objects: make(map[string]object),
	}
}

type cache struct {
	objects map[string]object
	sync.RWMutex
}

type object struct {
	name      string
	dbColumns map[string]string
	editable  map[string]bool
}

func (obj *object) addFields(field reflect.StructField) {
	// finding object fields that have a tag: `editable`
	if tag, ok := field.Tag.Lookup("editable"); ok && (tag == "true" || tag == "yes") {
		obj.editable[field.Name] = true
	}

	// collecting `db` fields
	if tag, ok := field.Tag.Lookup("db"); ok && (tag != "-") {
		obj.dbColumns[field.Name] = tag
	}
}

func (m *cache) inspectObject(obj interface{}) (object, error) {
	if obj == nil {
		return object{}, errors.New("object is nil")
	}

	var objInfo reflect.Value

	// inspecting passed object
	switch v := reflect.ValueOf(obj); v.Kind() {
	case reflect.Struct:
		objInfo = v
	case reflect.Ptr:
		objInfo = v.Elem()
	}

	// checking cache first
	m.RLock()
	cached, ok := m.objects[objInfo.Type().Name()]
	m.RUnlock()

	// if found, returning cached object
	if ok {
		return cached, nil
	}

	// initializing object
	o := object{
		name:      objInfo.Type().Name(),
		dbColumns: make(map[string]string),
		editable:  make(map[string]bool),
	}

	// collecting editable fields
	for i := 0; i < objInfo.NumField(); i++ {
		field := objInfo.Type().Field(i)

		// processing the object struct itself
		o.addFields(field)

		switch field.Type.Kind() {
		case reflect.Struct:
			for j := 0; j < field.Type.NumField(); j++ {
				o.addFields(field.Type.Field(j))
			}
		}
	}

	// caching metadata object
	m.Lock()
	m.objects[objInfo.Type().Name()] = o
	m.Unlock()

	return o, nil
}

// Check checks whether any of the given field names
// are not among a list of editable fields
func Check(obj interface{}, names ...string) error {
	metadata, err := instance.inspectObject(obj)
	if err != nil {
		return errors.Wrap(err, "failed to inspect object")
	}

	return checkWithMetadata(metadata, names...)
}

func checkWithMetadata(metadata object, names ...string) error {
	for _, name := range names {
		if _, ok := metadata.editable[name]; !ok {
			return errors.Errorf("field `%s` is protected and not editable", name)
		}
	}

	return nil
}

// ListEditable returns a list of editable fields for a given object
func ListEditable(obj interface{}) []string {
	metadata, err := instance.inspectObject(obj)
	if err != nil {
		panic(errors.Wrap(err, "failed to inspect object"))
	}

	keys := make([]string, len(metadata.editable))
	i := 0
	for k := range metadata.editable {
		keys[i] = k
		i++
	}

	return keys
}

// ProcureDBChangesFromChangelog produces a map of changes
// based on a given object and a changelog
// NOTE: this function does not check whether any of the changed fields are protected
// NOTE: returned keys are the data column names mapped with `db`
// TODO: move to data package
func ProcureDBChangesFromChangelog(obj interface{}, changelog diff.Changelog) (changes map[string]interface{}, err error) {
	metadata, err := instance.inspectObject(obj)
	if err != nil {
		panic(errors.Wrap(err, "failed to inspect object"))
	}

	// initializing result map
	changes = make(map[string]interface{})
	changedFields := make([]string, 0)

	// building changelist
	for _, c := range changelog {
		// changed object field name
		field := c.Path[0]

		// TODO: find a better solution instead of hardcoding
		// accommodating for nested fields
		// NOTE: an ugly exclusive fix for a Time type
		if l := len(c.Path); l > 1 {
			// if the last nested name is not `Time`, then skip this field
			if c.Path[l-1] != "Time" {
				continue
			}

			// otherwise, set field to the pre-last value
			field = c.Path[l-2]
		}

		// checking whether this field has a data
		// column name mapped
		dbColumn, ok := metadata.dbColumns[field]
		if !ok {
			continue
		}

		// mapping data column to a new value
		changes[dbColumn] = c.To

		// appending a changed object field name for further checking
		changedFields = append(changedFields, field)
	}

	return changes, nil
}

// DBColumnsFrom returns a slice of data field
// names declared via `db` tag
// NOTE: will only include fields with explicit `db` tag
func DBColumnsFrom(obj interface{}) (columns []string) {
	metadata, err := instance.inspectObject(obj)
	if err != nil {
		panic(errors.Wrap(err, "failed to inspect object"))
	}

	// collecting tagged names
	columns = make([]string, 0)
	for _, columnName := range metadata.dbColumns {
		columns = append(columns, columnName)
	}

	return columns
}
