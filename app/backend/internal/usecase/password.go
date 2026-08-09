package usecase

type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(hash, password string) (bool, error)
}
