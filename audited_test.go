package audited_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/iwarapter/audited"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      logger.Info, // Log level
			Colorful:      true,        // Disable color
		},
	)
	var err error
	db, err = gorm.Open(postgres.Open("host=localhost port=5432 user=postgres dbname=postgres password=postgres sslmode=disable"), &gorm.Config{
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
	Expiry time.Time
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
			return d.Set("audited:current_user", "create-test")
		}).Create(&user).Error)
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
			return d.Set("audited:current_user", "update-test")
		}).Save(&user).Error)
		require.NotNil(t, user.CreatedBy)
		assert.Equal(t, "create-test", *user.CreatedBy)
		require.NotNil(t, user.UpdatedBy)
		assert.Equal(t, "update-test", *user.UpdatedBy)
		assert.Equal(t, birth, user.DOB)
	})
	t.Run("an audited association has created_by set and updated_by", func(t *testing.T) {
		product := User{
			Name: "old joe",
			CreditCards: []CreditCard{
				{
					Number: "987654321",
				},
			},
		}
		require.NoError(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return d.Set("audited:current_user", "create-test")
		}).Create(&product).Error)
		assert.Equal(t, "create-test", *product.CreatedBy)
		assert.Nil(t, product.UpdatedBy)

		require.NotNil(t, product.CreditCards[0].CreatedBy)
		assert.Equal(t, "create-test", *product.CreditCards[0].CreatedBy)
		assert.Nil(t, product.CreditCards[0].UpdatedBy)

		product.DOB = time.Now()

		require.NoError(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return d.Set("audited:current_user", "update-test")
		}).Save(&product).Error)

		require.NotNil(t, product.UpdatedBy)
		assert.Equal(t, "update-test", *product.UpdatedBy)
	})
	t.Run("an audited association has created_by set for multiple", func(t *testing.T) {
		product := User{
			Name: "old joe blogs",
			CreditCards: []CreditCard{
				{
					Number: "1212121212",
				},
				{
					Number: "2121212121",
				},
			},
		}
		require.NoError(t, db.Scopes(func(d *gorm.DB) *gorm.DB {
			return d.Set("audited:current_user", "create-test")
		}).Create(&product).Error)
		assert.Equal(t, "create-test", *product.CreatedBy)
		assert.Nil(t, product.UpdatedBy)

		require.NotNil(t, product.CreditCards[0].CreatedBy)
		assert.Equal(t, "create-test", *product.CreditCards[0].CreatedBy)
		assert.Nil(t, product.CreditCards[0].UpdatedBy)
		require.NotNil(t, product.CreditCards[1].CreatedBy)
		assert.Equal(t, "create-test", *product.CreditCards[1].CreatedBy)
		assert.Nil(t, product.CreditCards[1].UpdatedBy)
	})
	t.Run("we can still save un-audited structs", func(t *testing.T) {
		user := &Example{Name: "example"}
		require.Nil(t, db.Create(&user).Error)
	})
}
