import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("import (", "import (\n\t\"crypto/aes\"\n\t\"crypto/cipher\"\n")

content = content.replace("type GraphConfig struct {\n\tAppID, AppSecret, RedirectURI, Version, WebhookFields, WebhookVerifyToken, StateFile string\n}", "type GraphConfig struct {\n\tAppID, AppSecret, RedirectURI, Version, WebhookFields, WebhookVerifyToken, StateFile, EncryptionKey string\n}")

encryption_funcs = """
func (g *GraphConnector) encryptToken(token string) string {
	if token == "" || g.cfg.EncryptionKey == "" {
		return token
	}
	key := sha256.Sum256([]byte(g.cfg.EncryptionKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return token
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return token
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return token
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)
	return "enc:" + hex.EncodeToString(ciphertext)
}

func (g *GraphConnector) decryptToken(cipherHex string) string {
	if !strings.HasPrefix(cipherHex, "enc:") || g.cfg.EncryptionKey == "" {
		return cipherHex
	}
	cipherHex = strings.TrimPrefix(cipherHex, "enc:")
	key := sha256.Sum256([]byte(g.cfg.EncryptionKey))
	ciphertext, err := hex.DecodeString(cipherHex)
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return ""
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
}
"""

content = content.replace("func (g *GraphConnector) loadState() error {", encryption_funcs + "\nfunc (g *GraphConnector) loadState() error {")

# Replace token saves
content = content.replace("g.db.Save(&database.MetaConnectionModel{AccountID: stateData.AccountID, UserToken: token.AccessToken})", "g.db.Save(&database.MetaConnectionModel{AccountID: stateData.AccountID, UserToken: g.encryptToken(token.AccessToken)})")
content = content.replace("g.db.Save(&database.MetaConnectionModel{AccountID: pending.AccountID, UserToken: token.AccessToken})", "g.db.Save(&database.MetaConnectionModel{AccountID: pending.AccountID, UserToken: g.encryptToken(token.AccessToken)})")

content = content.replace("AccessToken: page.AccessToken,", "AccessToken: g.encryptToken(page.AccessToken),")
# And decrypt when loading
content = content.replace("g.sessions[accountID].UserToken = model.UserToken", "g.sessions[accountID].UserToken = g.decryptToken(model.UserToken)")
content = content.replace("AccessToken: model.AccessToken,", "AccessToken: g.decryptToken(model.AccessToken),")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
