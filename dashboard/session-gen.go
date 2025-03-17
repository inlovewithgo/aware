package dashboard

import (
    "crypto/rand"
    "encoding/base64"
)

func generateSessionSecret() (string, error) {
    b := make([]byte, 32)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return base64.StdEncoding.EncodeToString(b), nil
}

func main() {
	generateSessionSecret()
}