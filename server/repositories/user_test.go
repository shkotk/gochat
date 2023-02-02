package repositories

import (
	"context"

	"github.com/shkotk/gochat/server/models"
	"github.com/sirupsen/logrus"
)

func (s *DBTestSuite) TestUser_Exists_CancelledContex_ReturnsError() {
	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := userRepository.Exists(ctx, "username")

	s.NotNil(err)
	s.Equal(context.Canceled, err)
}

func (s *DBTestSuite) TestUser_Exists_PopulatedUsersTable_ReturnsExpectedResult() {
	s.testDB.Create([]models.User{
		{
			Username:     "stanley",
			PasswordHash: "somehash",
		},
		{
			Username:     "kevin",
			PasswordHash: "otherhash",
		},
	})

	type testCase struct {
		username       string
		expectedResult bool
	}

	cases := []testCase{
		{"stanley", true},
		{"michael", false},
		{"kevin", true},
	}

	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	for _, test := range cases {
		s.Run(test.username, func() {
			actual, err := userRepository.Exists(context.Background(), test.username)

			s.Nil(err)
			s.Equal(test.expectedResult, actual, "got wrong result for %s", test.username)
		})
	}
}

func (s *DBTestSuite) TestUser_Create_CancelledContext_ReturnsError() {
	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := userRepository.Create(ctx, models.User{})

	s.Equal(context.Canceled, err)
}

func (s *DBTestSuite) TestUser_Create_InvalidUser_ReturnsError() {
	s.testDB.Create(&models.User{
		Username:     "shouldBeUnique",
		PasswordHash: "somehash",
	})

	type testCase struct {
		label         string
		user          models.User
		expectedError string
	}

	cases := []testCase{
		{
			label:         "missing username and password hash",
			user:          models.User{},
			expectedError: `null value in column "username" of relation "users" violates not-null constraint`,
		},
		{
			label:         "missing password hash",
			user:          models.User{Username: "username"},
			expectedError: `null value in column "password_hash" of relation "users" violates not-null constraint`,
		},
		{
			label:         "missing username",
			user:          models.User{PasswordHash: "somehash"},
			expectedError: `null value in column "username" of relation "users" violates not-null constraint`,
		},
		{
			label:         "username duplicate",
			user:          models.User{Username: "shouldBeUnique", PasswordHash: "somehash"},
			expectedError: `duplicate key value violates unique constraint`,
		},
	}

	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	for _, test := range cases {
		s.Run(test.label, func() {
			err := userRepository.Create(context.Background(), test.user)

			s.ErrorContains(err, test.expectedError)
		})
	}
}

func (s *DBTestSuite) TestUser_Create_ValidUser_AddsRecord() {
	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	err := userRepository.Create(
		context.Background(),
		models.User{
			Username:     "someusername",
			PasswordHash: "somehash",
		})

	s.Nil(err)

	users := []models.User{}
	s.testDB.Take(&users)
	s.Len(users, 1)
	s.Equal("someusername", users[0].Username)
	s.Equal("somehash", users[0].PasswordHash)
}

func (s *DBTestSuite) TestUser_Get_CancelledContext_ReturnsError() {
	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := userRepository.Get(ctx, "username")

	s.Equal(context.Canceled, err)
}

func (s *DBTestSuite) TestUser_Get_PopulatedUsersTable_ReturnsExpectedResult() {
	s.testDB.Create([]models.User{
		{
			Username:     "stanley",
			PasswordHash: "somehash",
		},
		{
			Username:     "kevin",
			PasswordHash: "otherhash",
		},
	})

	type testCase struct {
		username       string
		expectedResult *models.User
	}

	cases := []testCase{
		{"stanley", &models.User{Username: "stanley", PasswordHash: "somehash"}},
		{"michael", nil},
		{"kevin", &models.User{Username: "kevin", PasswordHash: "otherhash"}},
	}

	userRepository := NewUserRepository(logrus.StandardLogger(), s.testDB)

	for _, test := range cases {
		s.Run(test.username, func() {
			actual, err := userRepository.Get(context.Background(), test.username)

			s.Nil(err)
			s.Equal(test.expectedResult, actual, "got wrong result for %s", test.username)
		})
	}
}
