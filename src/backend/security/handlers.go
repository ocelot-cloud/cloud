package security

import (
	"github.com/ocelot-cloud/shared/utils"
	"github.com/ocelot-cloud/shared/validation"
	"net/http"
	"ocelot/backend/tools"
	"strconv"
)

var Logger = tools.Logger

func InitializeUserModule() {
	RegisterRoutes([]Route{
		{tools.UsersListPath, ListUsersHandler, Admin},
		{tools.UsersCreatePath, CreateUserHandler, Admin},
		{tools.UsersDeletePath, DeleteUserHandler, Admin},
		{tools.UsersLogoutPath, LogoutHandler, User},
		{tools.ChangePasswordPath, ChangePasswordHandler, User},
	})
}

type UserDto struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := UserRepo.ListUsers()
	if err != nil {
		Logger.Error("Failed to list users: %v", err)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
	}

	var userDtos []UserDto
	for _, user := range users {
		var role string
		if user.IsAdmin {
			role = "admin"
		} else {
			role = "user"
		}

		userDtos = append(userDtos, UserDto{
			Id:   strconv.Itoa(user.Id),
			Name: user.Name,
			Role: role,
		})
	}

	utils.SendJsonResponse(w, userDtos)
}

type Credentials struct {
	Username string `json:"username" validate:"user_name"`
	Password string `json:"password" validate:"password"`
}

func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	creds, err := validation.ReadBody[Credentials](w, r)
	if err != nil {
		return
	}

	if UserRepo.DoesUserExist(creds.Username) {
		Logger.Info("user already exists: %s", creds.Username)
		http.Error(w, "user already exists", http.StatusConflict)
		return
	}

	err = UserRepo.CreateUser(creds.Username, creds.Password, false)
	if err != nil {
		Logger.Error("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	tools.WriteResponse(w, "user created")
}

func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userIdString, err := validation.ReadBody[tools.NumberString](w, r)
	if err != nil {
		return
	}

	userId, ok := strconv.Atoi(userIdString.Value)
	if ok != nil {
		Logger.Error("Failed to convert user id to integer: %v", err)
		http.Error(w, "Failed to convert user id to integer", http.StatusBadRequest)
		return
	}

	context, err := GetAuthFromContext(w, r)
	if err != nil {
		return
	}

	if context.UserId == userId {
		Logger.Error("an admin can not delete his own account")
		http.Error(w, "an admin can not delete his own account", http.StatusUnauthorized)
		return
	}

	err = UserRepo.DeleteUser(userId)
	if err != nil {
		Logger.Error("Failed to delete user: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	tools.WriteResponse(w, "user deleted")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	auth, err := GetAuthFromContext(w, r)
	if err != nil {
		return
	}

	err = UserRepo.Logout(auth.User)
	if err != nil {
		Logger.Error("Failed to logout: %v", err)
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	tools.WriteResponse(w, "logged out")
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	auth, err := GetAuthFromContext(w, r)
	if err != nil {
		return
	}

	newPasswordString, err := validation.ReadBody[tools.PasswordString](w, r)
	if err != nil {
		return
	}

	err = UserRepo.ChangePassword(auth.UserId, newPasswordString.Value)
	if err != nil {
		Logger.Error("Failed to logout: %v", err)
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	tools.WriteResponse(w, "logged out")
}
