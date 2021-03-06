package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"strconv"
	"strings"
	"time"
)

// The User model represents people who have accounts
type User struct {
	Id             int64  `json:"id"`         // The identifier of the user
	FirstName      string `json:"firstName"`  // The first name of the user
	LastName       string `json:"lastName"`   // The last name of the user
	Email          string `json:"email"`      // The email address of the user (indexed)
	HashedPassword string `json:"-"`          // The bcrypted password of the user
	StripeId       string `json:"-"`          // The id of the user with Stripe's API
	PictureUrl     string `json:"pictureUrl"` // The URL to user's picture

	Active    bool        `json:"active"`    // True if this entity has not been soft deleted
	CreatedAt time.Time   `json:"createdAt"` // The time when this user was created
	UpdatedAt time.Time   `json:"updatedAt"` // The time when this user was last updated
	DeletedAt pq.NullTime `json:"-"`         // The time when this user was soft deleted
}

const (
	TABLE_NAME_USER = "users"

	FIELD_USER_STRIPE_ID = "stripe_id"

	SQL_CREATE_TABLE_USER = `
		CREATE TABLE IF NOT EXISTS ` + TABLE_NAME_USER + `(
			id				BIGSERIAL			PRIMARY KEY,
			first_name		VARCHAR(100)		NOT NULL,
			last_name		VARCHAR(100)		NOT NULL,
			email			VARCHAR(255)		UNIQUE NOT NULL,
			hashed_password	VARCHAR(255)		NOT NULL,
			stripe_id		VARCHAR(255)		NOT NULL,
			picture_url		VARCHAR(511),

			active			BOOLEAN				NOT NULL,
			created_at		TIMESTAMPTZ			NOT NULL,
			updated_at		TIMESTAMPTZ			NOT NULL,
			deleted_at		TIMESTAMPTZ
		);
	`
	SQL_CREATE_NEW_USER = `
		INSERT INTO ` + TABLE_NAME_USER + `
		(first_name, last_name, email, hashed_password, stripe_id, picture_url, active, created_at, updated_at) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;
	`
	SQL_SELECT_USER_BY_ID = `
		SELECT * FROM ` + TABLE_NAME_USER + ` WHERE (id = $1);
	`
	SQL_SELECT_USER_BY_EMAIL = `
		SELECT * FROM ` + TABLE_NAME_USER + ` WHERE (email = $1);
	`
	SQL_SELECT_USERS = `
		SELECT * FROM ` + TABLE_NAME_USER + ` OFFSET $1 LIMIT $2;
	`
	SQL_UPDATE_USER = `
		UPDATE ` + TABLE_NAME_USER + ` SET %s WHERE (id = $1);;
	`
)

// Fills user with data from a db row
func (u User) populateFromRow(row *sql.Row) error {
	// Scan for member fields
	Debug("Populate from row ", *row)
	return row.Scan(&u.Id, &u.FirstName, &u.LastName, &u.Email, &u.HashedPassword, &u.StripeId, &u.PictureUrl, &u.Active, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
}

// Creates the User table if it doesn't already exist
func CreateUserTable(db *sql.DB) error {
	_, err := db.Exec(SQL_CREATE_TABLE_USER)
	return err
}

// Gets a User from the database by id
func GetUser(
	db Queryable,
	id int64,
) (*User, error) {
	rows, err := db.Query(SQL_SELECT_USER_BY_ID, id)
	if err != nil {
		return nil, PUBERR_ENTITY_NOT_FOUND
	}
	// Read the rows
	defer rows.Close()
	var user User
	for rows.Next() {
		err = rows.Scan(&user.Id, &user.FirstName, &user.LastName, &user.Email, &user.HashedPassword, &user.StripeId, &user.PictureUrl, &user.Active, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt)
		if err != nil {
			return nil, err
		} else {
			return &user, nil
		}
	}
	// We didn't find any users
	return nil, PUBERR_ENTITY_NOT_FOUND
}

// Gets a User from the database by id
func GetUsers(
	db Queryable,
	offset int,
	limit int,
) ([]*User, error) {
	rows, err := db.Query(SQL_SELECT_USERS, offset, limit)
	if err != nil {
		return nil, PUBERR_ENTITY_NOT_FOUND
	}
	// Read the rows
	defer rows.Close()
	users := make([]*User, 0, limit)
	for rows.Next() {
		var newUser User
		err = rows.Scan(&newUser.Id, &newUser.FirstName, &newUser.LastName, &newUser.Email, &newUser.HashedPassword, &newUser.StripeId, &newUser.PictureUrl, &newUser.Active, &newUser.CreatedAt, &newUser.UpdatedAt, &newUser.DeletedAt)
		if err != nil {
			return nil, err
		} else {
			users = append(users, &newUser)
		}
	}
	// We didn't find any users
	return users, nil
}

// Finds a User by email
func FindUserByEmail(
	db Queryable,
	email string,
) (*User, error) {
	rows, err := db.Query(SQL_SELECT_USER_BY_EMAIL, email)
	if err != nil {
		return nil, PUBERR_ENTITY_NOT_FOUND
	}
	// Read the rows
	defer rows.Close()
	var newUser User
	for rows.Next() {
		err = rows.Scan(&newUser.Id, &newUser.FirstName, &newUser.LastName, &newUser.Email, &newUser.HashedPassword, &newUser.StripeId, &newUser.PictureUrl, &newUser.Active, &newUser.CreatedAt, &newUser.UpdatedAt, &newUser.DeletedAt)
		if err != nil {
			return nil, err
		} else {
			return &newUser, nil
		}
	}
	// We didn't find any users
	return nil, PUBERR_ENTITY_NOT_FOUND
}

// Creates a new User in the database; returns the id of the new user
func CreateNewUser(
	db Queryable, // The database
	FirstName string, // The first name of the user
	LastName string, // The last name of the user
	Email string, // The email address of the user (indexed)
	HashedPassword string, // The bcrypted password of the user
	StripeId string, // The id of the user with Stripe's API
	PictureUrl string, // The URL to user's picture
) (int64, error) {
	var (
		id  int64
		now = time.Now()
	)
	err := db.QueryRow(SQL_CREATE_NEW_USER, FirstName, LastName, Email, HashedPassword, StripeId, PictureUrl, true, now, now).Scan(&id)
	if err != nil {
		// Check if the issue is email related
		if strings.Contains(err.Error(), "violates unique constraint \"users_email_key\"") {
			return -1, PUBERR_USER_CREATION_FAILED_EMAIL_TAKEN
		} else {
			return -1, err
		}
	} else {
		return id, nil
	}
}

// Updates a specific set of fields of a user
func UpdateUserFields(
	db Queryable, // The database
	id int64, // The id of the user being updated
	keyVals map[string]interface{}, // Field deltas
) error {
	if len(keyVals) < 1 {
		return nil
	}
	// Ensure "updated_at" is accurate
	keyVals["updated_at"] = time.Now()

	var (
		updates bytes.Buffer
		values  = make([]interface{}, 1, (len(keyVals) + 1))
		i       = 0
	)
	// Add the id as the first query param
	values[0] = id
	// Built the SET section of the quuery
	for fieldName, fieldVal := range keyVals {
		if i > 0 {
			updates.WriteString(", ")
		}
		updates.WriteString(fieldName)
		updates.WriteString(" = $")
		updates.WriteString(strconv.Itoa(i + 2))
		values = append(values, fieldVal)
		i = i + 1
	}
	// Execute the query
	_, err := db.Exec(fmt.Sprintf(SQL_UPDATE_USER, updates.String()), values...)
	return err
}
