package storage

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"gofermart/internal/config"
	"gofermart/internal/models"
)

type Repository interface {
	UserRegistered(login string) *models.User
	RegisterUser(model *models.User) error
	LoginUser(login string, password string) *models.User
	GetUser(password string) *models.User
	GetOrder(order int) *models.Order
	SetOrder(*models.Order) error
	GetOrders(id uint64) []models.Order
	SetWithdraw(*models.Balance) error
	GetWithdraws(id uint64) []models.Balance
	SetAccrual(id int, status string, accrual float64) error
	GetOrdersByStatus() []models.Order
}

type repository struct {
	db *gorm.DB
}

type DB struct {
	Repo Repository
}

func NewDB() *DB {
	repo := NewRepository(config.GetConfigDBAddress())

	return &DB{
		Repo: repo,
	}
}

func NewRepository(dns string) Repository {
	db, err := gorm.Open(postgres.Open(dns), &gorm.Config{})
	if err != nil {
		log.Fatal("Gorm repository failed %w", err.Error())
	}
	if exist := db.Migrator().HasTable(&models.User{}); !exist {
		db.Migrator().CreateTable(&models.User{})
	}
	if exist := db.Migrator().HasTable(&models.Order{}); !exist {
		db.Migrator().CreateTable(&models.Order{})
	}
	if exist := db.Migrator().HasTable(&models.Balance{}); !exist {
		db.Migrator().CreateTable(&models.Balance{})
	}

	return &repository{db}
}

func (r *repository) UserRegistered(login string) *models.User {
	model := &models.User{}
	if err := r.db.Limit(1).Find(model, "login = ?", login).Error; err != nil {

		return nil
	}

	return model
}

func (r *repository) RegisterUser(m *models.User) error {
	if err := r.db.Create(m).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) LoginUser(login string, password string) *models.User {
	model := &models.User{}
	if err := r.db.Limit(1).Find(model, "login = ? AND password = ?", login, password).Error; err != nil {

		return nil
	}

	return model
}

func (r *repository) GetUser(password string) *models.User {
	model := &models.User{}
	if err := r.db.Limit(1).Find(model, "password = ?", password).Error; err != nil {

		return nil
	}

	return model
}

func (r *repository) GetOrder(order int) *models.Order {
	model := &models.Order{}
	if err := r.db.Limit(1).Find(model, "order_number = ?", order).Error; err != nil {

		return nil
	}

	return model
}

func (r *repository) SetOrder(m *models.Order) error {
	if err := r.db.Create(m).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) GetOrders(id uint64) []models.Order {
	orders := []models.Order{}
	r.db.Where("user_id = ?", id).Order("created_at desc").Find(&orders)

	return orders
}

func (r *repository) SetWithdraw(m *models.Balance) error {
	if err := r.db.Create(m).Error; err != nil {
		return err
	}

	return nil
}

func (r *repository) GetWithdraws(id uint64) []models.Balance {
	balances := []models.Balance{}
	r.db.Where("user_id = ?", id).Order("updated_at desc").Find(&balances)

	return balances
}

func (r *repository) SetAccrual(orderNumber int, status string, accrual float64) error {
	model := &models.Order{}
	if err := r.db.Limit(1).Find(model, "order_number  = ?", orderNumber).Error; err != nil {

		return err
	}
	model.Accrual = accrual
	model.Status = status
	r.db.Save(&model)

	return nil
}

func (r *repository) GetOrdersByStatus() []models.Order {
	orders := []models.Order{}
	r.db.Where("status IN ?", []string{"NEW", "REGISTERED", "PROCESSING"}).Find(&orders)

	return orders
}
