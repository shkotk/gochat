package repositories

import (
	"fmt"
	"log"
	"regexp"
	"testing"

	"github.com/google/uuid"
	"github.com/shkotk/gochat/server/models"
	"github.com/shkotk/gochat/server/test"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This test suite is responsible for setting up and tearing down DB for tests.
type DBTestSuite struct {
	suite.Suite
	setupDB    *gorm.DB
	testDB     *gorm.DB
	testDBName string
}

func (s *DBTestSuite) SetupSuite() {
	var err error
	testCfg := test.LoadConfig("../.test.env")

	if s.setupDB, err = gorm.Open(postgres.Open(testCfg.DBConnString)); err != nil {
		panic(err)
	}

	s.testDBName = "gochat_test_db_" + uuid.NewString()
	err = s.setupDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, s.testDBName)).Error
	if err != nil {
		panic(err)
	}

	dbnameRegexp := regexp.MustCompile(`dbname=\S*`)
	testDbConnString := dbnameRegexp.ReplaceAllString(
		testCfg.DBConnString,
		"dbname="+s.testDBName,
	)
	if s.testDB, err = gorm.Open(postgres.Open(testDbConnString)); err != nil {
		panic(err)
	}

	if err = s.testDB.AutoMigrate(models.User{}); err != nil {
		panic(err)
	}
}

func (s *DBTestSuite) TearDownTest() {
	err := s.testDB.Exec(`TRUNCATE TABLE "users"`).Error
	if err != nil {
		panic(err)
	}
}

func (s *DBTestSuite) TearDownSuite() {
	testSQLDB, err := s.testDB.DB()
	if err != nil {
		log.Println(err)
	}

	if err = testSQLDB.Close(); err != nil {
		log.Println(err)
	}

	err = s.setupDB.Exec(fmt.Sprintf(`DROP DATABASE "%s"`, s.testDBName)).Error
	if err != nil {
		log.Println(err)
	}

	setupSQLDB, err := s.setupDB.DB()
	if err != nil {
		log.Println(err)
	}

	if err = setupSQLDB.Close(); err != nil {
		log.Println(err)
	}
}

func TestDBSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
