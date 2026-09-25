package services

import (
	"errors"
	"log"
	"mini-crypto-wallet-api/db_conn"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/repositories"

	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// ErrUserAlreadyExists is returned by CreateUser when the requested
// username or email is already taken. Handlers should map this to
// HTTP 409 + errors.ErrCodeUserAlreadyExists rather than a generic 500 -
// this is a routine, expected client error, not a server failure.
// See docs/AUDIT.md S7.
var ErrUserAlreadyExists = errors.New("username or email already exists")

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
	// Reject duplicate username/email up front with a specific, expected
	// error rather than letting it surface as a generic DB/500 error from
	// the unique constraint violation at insert time. See docs/AUDIT.md S7.
	if _, err := s.userRepo.GetUserByUsername(req.Username); err == nil {
		return nil, ErrUserAlreadyExists
	}
	if _, err := s.userRepo.GetUserByEmail(req.Email); err == nil {
		return nil, ErrUserAlreadyExists
	}

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
		log.Println("⚠️ failed to create user record:", err)
		return nil, errors.New("failed to create user")
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
		log.Println("⚠️ failed to create wallet during user creation:", err)
		return nil, errors.New("failed to create user")
	}

	if commitDB := tx.Commit(); commitDB.Error != nil {
		// Log the real DB error for operators; don't return it to the
		// caller, since it can contain internal details (table/constraint
		// names, driver-specific text). See docs/AUDIT.md S9.
		log.Println("⚠️ failed to commit user creation transaction:", commitDB.Error)
		return nil, errors.New("failed to create user")
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
