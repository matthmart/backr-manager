package bcrypt

import "testing"

func TestGenerateAndCheckPassword(t *testing.T) {
	pwd, err := GeneratePassword()

	if err != nil {
		t.Fatalf("err should be nil. got=%v", err)
	}
	if pwd.Plain == "" {
		t.Fatal("Plain should not be empty")
	}
	if pwd.Hashed == "" {
		t.Fatal("Hashed should not be empty")
	}

	err = CompareHashAndPassword(pwd.Hashed, pwd.Plain)
	if err != nil {
		t.Fatalf("Plain and Hashed passwords should match with each other")
	}

}
