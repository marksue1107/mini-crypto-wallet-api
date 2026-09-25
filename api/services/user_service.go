package services

import (
	"errors"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo     repositories.IUser
	walletRepo   repositories.IWallet
	currencyRepo repositories.ICurrency
}

func NewUserService(userRepo repositories.IUser, walletRepo repositories.IWallet, currencyRepo repositories.ICurrency) *UserService {
	return &UserService{
		userRepo:     userRepo,
		walletRepo:   walletRepo,
		currencyRepo: currencyRepo,
	}
}

// CreateUser creates a new user from DTO and returns the created user model
// Accepts DTO to decouple HTTP layer from database layer
func (s *UserService) CreateUser(req *models.UserCreateRequest) (*models.User, error) {
	// Hash password from request
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create database model from DTO
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	// 使用事務確保用戶和錢包創建的原子性
	tx := db_conn.Conn_DB.MasterDB.Begin()
	// committed tracks whether we reached tx.Commit(). Every return path
	// below must roll back if we didn't get there, or the connection leaks
	// in an idle-in-transaction state (same bug class as, and fixed the
	// same way as, docs/AUDIT_REMEDIATION_PLAN.md finding N2).
	committed := false
	defer func() {
		if committed {
			return
		}
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
		tx.Rollback()
	}()

	// CreateUser must run inside tx: previously it used the repository's
	// non-transactional connection and was committed (autocommit) the
	// instant it ran, so if wallet creation failed afterwards the rollback
	// below could never undo it - leaving an orphaned user with no wallet
	// (docs/AUDIT_REMEDIATION_PLAN.md finding N3).
	if err := s.userRepo.CreateUser(user, tx); err != nil {
		return nil, err
	}

	// 獲取預設幣種（USDT），如果不存在則使用第一個幣種
	defaultCurrency, err := s.currencyRepo.GetCurrencyByCode("USDT")
	if err != nil {
		// 如果 USDT 不存在，嘗試獲取第一個幣種
		currencies, err := s.currencyRepo.GetAllCurrencies()
		if err != nil || len(currencies) == 0 {
			return nil, errors.New("no currency available")
		}
		defaultCurrency = &currencies[0]
	}

	wallet := &models.Wallet{
		UserID:     user.ID,
		CurrencyID: defaultCurrency.ID,
		Balance:    decimal.NewFromInt(1000),
	}

	if err := s.walletRepo.CreateWallet(wallet, tx); err != nil {
		return nil, err
	}

	if commitDB := tx.Commit(); commitDB.Error != nil {
		return nil, commitDB.Error
	}
	committed = true

	return user, nil
}

func (s *UserService) Login(username, password string) (*models.User, error) {
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	return user, nil
}
