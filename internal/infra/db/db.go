// Copyright The MatrixHub Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	case "postgres":
		dialector = gpostgres.Open(config.DSN)
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
