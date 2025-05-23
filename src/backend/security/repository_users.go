package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ocelot-cloud/shared/utils"
	"golang.org/x/crypto/bcrypt"
	"ocelot/backend/apps/common"
	"ocelot/backend/tools"
	"sync"
	"time"
)

var (
	UserRepo UserRepository = &UserRepositoryImpl{}
)

type UserRepositoryImpl struct{}

func (r *UserRepositoryImpl) IsExpired(cookieValue string) (bool, error) {
	var cookieExpirationDateString string
	err := common.DB.QueryRow("SELECT cookie_expiration_date FROM users WHERE hashed_cookie_value = $1", hashCookie(cookieValue)).Scan(&cookieExpirationDateString)
	if err != nil {
		tools.Logger.Error("Failed to fetch cookie expiration date: %v", err)
		return false, fmt.Errorf("failed to fetch cookie expiration date")
	}

	cookieExpirationDate, err := time.Parse(time.RFC3339, cookieExpirationDateString)
	if err != nil {
		tools.Logger.Error("Failed to parse cookie expiration date: %v", err)
		return false, errors.New("failed to parse cookie expiration date")
	}

	return time.Now().After(cookieExpirationDate), nil
}

// Not meant for production
func (r *UserRepositoryImpl) GetAllUsersFullInfo() ([]tools.UserFullInfo, error) {
	rows, err := common.DB.Query("SELECT user_id, user_name, hashed_password, hashed_cookie_value, cookie_expiration_date, is_admin FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to get all users: %v", err)
	}
	defer utils.Close(rows)

	var users []tools.UserFullInfo
	for rows.Next() {
		var user tools.UserFullInfo
		var cv, ce *string
		err = rows.Scan(&user.Id, &user.UserName, &user.HashedPassword, &cv, &ce, &user.IsAdmin)
		user.HashedCookieValue = cv
		user.CookieExpirationDate = ce
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %v", err)
		}
		if user.UserName != "admin" {
			users = append(users, user)
		}
	}
	return users, nil
}

// Not meant for production
func (r *UserRepositoryImpl) DeleteUsersAndAddUsersFullInfo(users []tools.UserFullInfo) error {
	tx, err := common.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	_, err = tx.Exec("DELETE FROM users WHERE user_name != $1", "admin")
	if err != nil {
		Logger.Error("Failed to delete users: %v", err)
		err2 := tx.Rollback()
		if err2 != nil {
			Logger.Error("Failed to rollback transaction: %v", err2)
		}
		return fmt.Errorf("failed to delete users: %v", err)
	}

	for _, user := range users {
		_, err = tx.Exec("INSERT INTO users (user_id, user_name, hashed_password, hashed_cookie_value, cookie_expiration_date, is_admin, did_accept_eula) VALUES ($1, $2, $3, $4, $5, $6, $7)",
			user.Id, user.UserName, user.HashedPassword, user.HashedCookieValue, user.CookieExpirationDate, user.IsAdmin, false)
		if err != nil {
			Logger.Error("Failed to insert user: %v", err)
			err2 := tx.Rollback()
			if err2 != nil {
				Logger.Error("Failed to rollback transaction: %v", err2)
			}
			return fmt.Errorf("failed to insert user: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}
	return nil
}

type UserRepository interface {
	CreateUser(user, password string, isAdmin bool) error
	GetUserId(user string) (int, error)
	IsPasswordCorrect(user, password string) bool
	DeleteUser(userId int) error
	SaveCookie(user, cookieValue string, cookieExpirationDate time.Time) error
	Logout(user string) error
	DoesUserExist(user string) bool
	GetAuthenticationViaCookie(cookieValue string) (*tools.Authorization, error)
	DoesAnyAdminUserExist() (bool, error)
	ChangePassword(userId int, newPassword string) error
	UpdateCookieExpirationDate(cookieValue string) error

	GenerateSecret(user string) (string, error)
	GetAssociatedCookieValueAndDeleteSecret(secret string) (string, error)
	ListUsers() ([]UserInfo, error)

	GetAllUsersFullInfo() ([]tools.UserFullInfo, error)
	DeleteUsersAndAddUsersFullInfo(users []tools.UserFullInfo) error
	IsExpired(value string) (bool, error)
}

func hashCookie(cookieValue string) string {
	hash := sha256.Sum256([]byte(cookieValue))
	return hex.EncodeToString(hash[:])
}

func (r *UserRepositoryImpl) DoesAnyAdminUserExist() (bool, error) {
	var exists bool
	err := common.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE is_admin = $1)", true).Scan(&exists)
	if err != nil {
		tools.Logger.Error("Failed to check if there is any admin user: %v", err)
		return false, fmt.Errorf("failed to check if there is any admin user")
	}
	return exists, nil
}

func (r *UserRepositoryImpl) CreateUser(user string, password string, isAdmin bool) error {
	hashedPassword, err := utils.SaltAndHash(password)
	if err != nil {
		return err
	}

	_, err = common.DB.Exec("INSERT INTO users (user_name, hashed_password, is_admin, did_accept_eula) VALUES ($1, $2, $3, $4)", user, hashedPassword, isAdmin, false)
	if err != nil {
		tools.Logger.Warn("Failed to create user: %v", err)
		return fmt.Errorf("failed to create user")
	}
	return nil
}

func (r *UserRepositoryImpl) IsPasswordCorrect(user string, password string) bool {
	var hashedPassword string
	err := common.DB.QueryRow("SELECT hashed_password FROM users WHERE user_name = $1", user).Scan(&hashedPassword)
	if err != nil {
		Logger.Info("Failed to fetch hashed password: %v", err)
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		Logger.Info("Password of user '%s' not matching: %v", user, err)
		return false
	}

	return true
}

func (r *UserRepositoryImpl) DeleteUser(userId int) error {
	_, err := common.DB.Exec("DELETE FROM users WHERE user_id = $1", userId)
	if err != nil {
		tools.Logger.Warn("Failed to delete user: %v", err)
		return fmt.Errorf("failed to delete user")
	}
	return nil
}

func (r *UserRepositoryImpl) SaveCookie(user string, cookieValue string, cookieExpirationDate time.Time) error {
	hashedCookie := hashCookie(cookieValue)
	_, err := common.DB.Exec("UPDATE users SET hashed_cookie_value = $1, cookie_expiration_date = $2 WHERE user_name = $3", hashedCookie, cookieExpirationDate.Format(time.RFC3339), user)
	if err != nil {
		tools.Logger.Warn("Failed to update cookie of user '%s': %v", user, err)
		return fmt.Errorf("failed to update cookie")
	}
	return nil
}

func (r *UserRepositoryImpl) Logout(user string) error {
	_, err := common.DB.Exec("UPDATE users SET hashed_cookie_value = $1, cookie_expiration_date = $2 WHERE user_name = $3", nil, "", user)
	if err != nil {
		tools.Logger.Error("Failed to delete cookie of user '%s': %v", user, err)
		return fmt.Errorf("failed to delete cookie")
	}
	return nil
}

func (r *UserRepositoryImpl) DoesUserExist(user string) bool {
	var exists bool
	err := common.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_name = $1)", user).Scan(&exists)
	if err != nil {
		tools.Logger.Error("Failed to check if user exists: %v", err)
		return false
	}
	return exists
}

func (r *UserRepositoryImpl) GetAuthenticationViaCookie(cookieValue string) (*tools.Authorization, error) {
	hashedCookieValue := hashCookie(cookieValue)
	var user, cookieExpirationDateString string
	var isAdmin, didAcceptEula bool
	var userId int
	err := common.DB.QueryRow("SELECT user_id, user_name, is_admin, did_accept_eula, cookie_expiration_date FROM users WHERE hashed_cookie_value = $1", hashedCookieValue).Scan(&userId, &user, &isAdmin, &didAcceptEula, &cookieExpirationDateString)
	if err != nil {
		tools.Logger.Info("Failed to fetch user data: %v", err)
		return nil, fmt.Errorf("failed to fetch user data")
	}

	err = r.UpdateCookieExpirationDate(cookieValue)
	if err != nil {
		Logger.Error("Failed to update cookie expiration date: %v", err)
		return nil, fmt.Errorf("failed to update cookie expiration date")
	}

	cookieExpirationDate, err := time.Parse(time.RFC3339, cookieExpirationDateString)
	if err != nil {
		tools.Logger.Error("Failed to parse cookie expiration date: %v", err)
		return nil, errors.New("failed to parse cookie expiration date")
	}

	return &tools.Authorization{
		UserId:               userId,
		User:                 user,
		IsAdmin:              isAdmin,
		DidAcceptEula:        didAcceptEula,
		CookieExpirationDate: cookieExpirationDate,
	}, nil
}

func (r *UserRepositoryImpl) ChangePassword(userId int, newPassword string) error {
	hashedNewPassword, err := utils.SaltAndHash(newPassword)
	if err != nil {
		return err
	}

	_, err = common.DB.Exec("UPDATE users SET hashed_password = $1 WHERE user_id = $2", hashedNewPassword, userId)
	if err != nil {
		tools.Logger.Error("Failed to update password of user with ID '%d': %v", userId, err)
		return fmt.Errorf("failed to update password")
	}
	return nil
}

var Secrets = sync.Map{}

func (r *UserRepositoryImpl) GenerateSecret(cookieValue string) (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret")
	}
	secret := hex.EncodeToString(randomBytes)
	Secrets.Store(secret, cookieValue)
	time.AfterFunc(60*time.Second, func() {
		Secrets.Delete(secret)
	})
	return secret, nil
}

func (r *UserRepositoryImpl) GetAssociatedCookieValueAndDeleteSecret(secret string) (string, error) {
	cookieValueObject, ok := Secrets.Load(secret)
	if !ok {
		return "", fmt.Errorf("secret not found")
	}
	cookieValue := cookieValueObject.(string)
	Secrets.Delete(secret)
	return cookieValue, nil
}

func (r *UserRepositoryImpl) GetUserId(user string) (int, error) {
	var userId int
	err := common.DB.QueryRow("Select user_id from users where user_name = $1", user).Scan(&userId)
	if err != nil {
		tools.Logger.Error("Failed to get user id: %v", err)
		return -1, err
	}
	return userId, nil
}

type UserInfo struct {
	Id      int
	Name    string
	IsAdmin bool
}

func (r *UserRepositoryImpl) ListUsers() ([]UserInfo, error) {
	rows, err := common.DB.Query("SELECT user_id, user_name, is_admin FROM users")
	if err != nil {
		tools.Logger.Error("Failed to list users: %v", err)
		return nil, fmt.Errorf("failed to list users")
	}
	defer utils.Close(rows)

	var users []UserInfo
	for rows.Next() {
		var userInfo UserInfo
		err = rows.Scan(&userInfo.Id, &userInfo.Name, &userInfo.IsAdmin)
		if err != nil {
			tools.Logger.Error("Failed to scan user: %v", err)
			return nil, fmt.Errorf("failed to list users")
		}
		users = append(users, userInfo)
	}
	return users, nil
}

func (r *UserRepositoryImpl) UpdateCookieExpirationDate(cookieValue string) error {
	_, err := common.DB.Exec("UPDATE users SET cookie_expiration_date = $1 WHERE hashed_cookie_value = $2", time.Now().Add(tools.CookieExpirationTime).Format(time.RFC3339), hashCookie(cookieValue))
	if err != nil {
		tools.Logger.Error("Failed to update cookie expiration date: %v", err)
		return fmt.Errorf("failed to update cookie expiration date")
	}
	return nil
}
