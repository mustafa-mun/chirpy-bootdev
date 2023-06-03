package bcrypt

import "golang.org/x/crypto/bcrypt"

func CreateHashedPassword(password string) (hashedPassword string, err error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	
	if err != nil {
		return "", err
	}

	hashedPassword = string(hashedBytes)
	return hashedPassword, nil
}

func CompareHashPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))

	if err != nil {
		return err
	}

	return nil
}