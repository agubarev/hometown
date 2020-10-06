package database

/*
var connPool *pgxpool.Pool

// Connection returns database singleton instance
func Connection() *pgxpool.Pool {
	// using a package global variable
	if connPool == nil {
		// checking whether it's called during `go test`
		testMode := flag.Lookup("test.v") != nil

		// describing database connPool config
		// NOTE: a quirky way to initialize blank config, otherwise
		// pgx panics with "panic: config must be created by ParseConfig"
		connConfig, err := pgxpool.ParseConfig("")
		if err != nil {
			panic(errors.Wrap(err, "failed to initialize database connection config"))
		}

		// telling viper to be aware of its environment
		viper.AutomaticEnv()

		host := strings.TrimSpace(viper.GetString(""))
		port := uint16(viper.GetInt(""))

		if host == "" {
			host = "127.0.0.1"
		}

		if port == 0 {
			port = 5432
		}

		// setting values from the system config
		connConfig.ConnConfig.Host =
		connConfig.ConnConfig.Port =
		connConfig.ConnConfig.Database =
		connConfig.ConnConfig.User =
		connConfig.ConnConfig.Password =

		if testMode {
			panic("DO NOT USE REGULAR CONNECTION WHEN TESTING")
			os.Exit(0)
		}

		// connecting to postgres database
		pool, err := pgxpool.ConnectConfig(context.Background(), connConfig)
		if err != nil {
			log.Fatalf("failed to connect to database: %s", err)
		}

		connPool = pool
	}

	return connPool
}

// ConnectionForTesting simply returns a database mysqlConn
func ConnectionForTesting() (conn *pgxpool.Pool, err error) {
	if !util.IsTestMode() {
		log.Fatal("TruncateTestDatabase() can only be called during testing")
		return nil, nil
	}

	// checking whether it's called during `go test`
	testMode := flag.Lookup("test.v") != nil
	if !testMode {
		log.Fatal("MUST RUN ONLY IN TEST MODE")
		os.Exit(0)
	}

	// describing database connPool config
	// NOTE: a quirky way to initialize blank config, otherwise
	// pgx panics with "panic: config must be created by ParseConfig"
	connConfig, err := pgxpool.ParseConfig("")
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize database connPool config"))
	}

	// telling viper to be aware of its environment
	viper.AutomaticEnv()

	ctx := context.Background()

	host := strings.TrimSpace(viper.GetString(""))
	port := uint16(viper.GetInt(""))

	if host == "" {
		host = "127.0.0.1"
	}

	if port == 0 {
		port = 5432
	}

	// setting values from the system config
	connConfig.ConnConfig.Host = host
	connConfig.ConnConfig.Port = port
	connConfig.ConnConfig.Database = viper.GetString("")
	connConfig.ConnConfig.User = viper.GetString("")
	connConfig.ConnConfig.Password = viper.GetString("")

	// connecting to postgres database
	conn, err = pgxpool.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatalf("failed to begin transaction: %s", err)
	}

	defer func() {
		if p := recover(); p != nil {
			err = errors.Wrap(err, "recovering from panic after TruncateDatabaseForTesting")
		}
	}()

	tables := []string{
	}

	// truncating tables
	for _, tableName := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`TRUNCATE TABLE "%s" RESTART IDENTITY CASCADE`, tableName)); err != nil {
			return nil, errors.Wrap(err, tx.Rollback(ctx).Error())
		}
	}

	if err := tx.Commit(ctx); err != nil {
		panic(err)
	}

	return conn, nil
}
*/
