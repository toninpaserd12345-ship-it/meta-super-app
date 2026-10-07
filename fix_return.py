import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

user_token_old = """			g.sessions[accountID].UserToken = g.decryptToken(model.UserToken)
			g.mu.Unlock()
			return model.UserToken, nil"""
user_token_new = """			decrypted := g.decryptToken(model.UserToken)
			g.sessions[accountID].UserToken = decrypted
			g.mu.Unlock()
			return decrypted, nil"""
content = content.replace(user_token_old, user_token_new)

find_page_old = """		if err := g.db.First(&model, "page_id = ?", pageID).Error; err == nil {
			page := graphPage{
				MetaPage: domain.MetaPage{
					ID:         model.PageID,
					Name:       model.Name,
					Category:   model.Category,
					PictureURL: model.PictureURL,
					Connected:  model.IsConnected,
					TokenReady: model.AccessToken != "",
				},
				AccessToken: g.decryptToken(model.AccessToken),
			}"""
find_page_new = """		if err := g.db.First(&model, "page_id = ?", pageID).Error; err == nil {
			decrypted := g.decryptToken(model.AccessToken)
			page := graphPage{
				MetaPage: domain.MetaPage{
					ID:         model.PageID,
					Name:       model.Name,
					Category:   model.Category,
					PictureURL: model.PictureURL,
					Connected:  model.IsConnected,
					TokenReady: decrypted != "",
				},
				AccessToken: decrypted,
			}"""
content = content.replace(find_page_old, find_page_new)

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
