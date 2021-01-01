package audited_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"

	"github.com/iwarapter/audited"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	dbURL := func(port nat.Port) string {
		return fmt.Sprintf("postgres://postgres:postgres@localhost:%s/postgres?sslmode=disable", port.Port())
	}
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:latest",
		ExposedPorts: []string{"5432/tcp"},
		Cmd:          []string{"postgres", "-c", "fsync=off"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_DB":       "postgres",
		},
		WaitingFor: wait.ForSQL("5432/tcp", "postgres", dbURL).Timeout(time.Second * 5),
	}
	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("could not start container: %v", err)
	}
	dbPort, err := postgresC.MappedPort(ctx, "5432/tcp")
	if err != nil {
		log.Fatalf("could not get mapped port: %v", err)
	}
	defer postgresC.Terminate(ctx)
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second,   // Slow SQL threshold
			LogLevel:      logger.Silent, // Log level
			Colorful:      false,         // Disable color
		},
	)

	db, err = gorm.Open(postgres.Open(fmt.Sprintf("host=localhost port=%s user=postgres dbname=postgres password=postgres sslmode=disable", dbPort.Port())), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

type User struct {
	gorm.Model
	audited.AuditedModel
	Name        string
	DOB         time.Time
	CreditCards []CreditCard
}

type CreditCard struct {
	gorm.Model
	audited.AuditedModel
	Number string
	UserID uint
}

type Example struct {
	gorm.Model
	Name string
}

func TestCreateUser(t *testing.T) {
	require.Nil(t, db.Use(&audited.Plugin{}))
	require.Nil(t, db.AutoMigrate(&User{}, &CreditCard{}, &Example{}))
	t.Run("an audited struct has created_by set", func(t *testing.T) {
		user := User{Name: "joe blogs"}
		require.Nil(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return db.Set("audited:current_user", "create-test")
		}).Save(&user).Error)
		require.NotNil(t, user.CreatedBy)
		assert.Equal(t, "create-test", *user.CreatedBy)
		assert.Nil(t, user.UpdatedBy)
	})
	t.Run("an audited struct has updated_by set", func(t *testing.T) {
		user := User{Name: "joe blogs"}
		require.Nil(t, db.First(&user).Error)
		birth := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		user.DOB = birth
		require.Nil(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return db.Set("audited:current_user", "update-test")
		}).Save(&user).Error)
		require.NotNil(t, user.CreatedBy)
		assert.Equal(t, "create-test", *user.CreatedBy)
		require.NotNil(t, user.UpdatedBy)
		assert.Equal(t, "update-test", *user.UpdatedBy)
		assert.Equal(t, birth, user.DOB)
	})
	t.Run("an audited association has created_by set", func(t *testing.T) {
		product := User{
			Name: "old joe",
			CreditCards: []CreditCard{
				{
					Number: "987654321",
				},
			},
		}
		require.Nil(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return db.Set("audited:current_user", "create-test")
		}).Save(&product).Error)
		assert.Equal(t, "create-test", *product.CreatedBy)
		assert.Nil(t, product.UpdatedBy)
		require.NotNil(t, product.CreditCards[0].CreatedBy)
		assert.Equal(t, "create-test", product.CreditCards[0].CreatedBy)
		assert.Nil(t, product.CreditCards[0].UpdatedBy)
	})
	t.Run("we can still save un-audited structs", func(t *testing.T) {
		user := &Example{Name: "example"}
		require.Nil(t, db.Create(&user).Error)
	})
}
