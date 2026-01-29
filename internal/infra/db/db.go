package db

import (
	"fmt"

	gmysql "gorm.io/driver/mysql"
	gpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(config Config) (*gorm.DB, error) {
	// utils.WaitExternalTcpServerReady(fmt.Sprintf("%v:%v", config.Host, config.Port))
	// add config validate
	var dialector gorm.Dialector
	switch config.Driver {
	case "mysql":
		dialector = gmysql.Open(config.DSN)
	case "postgres", "kingbase", "opengauss":
		dialector = gpostgres.Open(config.DSN)
	case "dm": // 由于达梦的数据库驱动目前只支持linux和windows, 在mac下无法运行, 因此注释掉dm. 如需使用, 可以解开import和dm.Open的注释
		// dialector = dm.Open(config.DSN)
	default:
		panic(fmt.Errorf("not support storage driver: %s", config.Driver))
	}

	db, err := gorm.Open(dialector, &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		return nil, err
	}

	if config.Migrate {
		if err = shouldMigrate(db, config.SQLPath, ""); err != nil {
			return nil, err
		}
	}
	if config.Debug {
		db = db.Debug()
	}

	return db, nil
}

// func getDBName(config Config) string {
// 	var dbname string
// 	switch config.Driver {
// 	case "mysql":
// 		cfg, err := mysql.ParseDSN(config.DSN)
// 		if err != nil {
// 			log.Infof("Error parsing mysql DSN:", err)
// 		} else {
// 			dbname = cfg.DBName
// 		}
// 	default:
// 		parsedURL, err := url.Parse(config.DSN)
// 		if err != nil {
// 			log.Infof("Error parsing %s DSN: %v", config.Driver, err)
// 		} else {
// 			dbname = strings.TrimPrefix(parsedURL.Path, "/")
// 		}
// 	}
// 	return dbname
// }
