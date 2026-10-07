import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

user_token_re = r'func \(g \*GraphConnector\) userToken\(accountID string\) \(string, error\) \{.*?\n\}'
user_token_new = """func (g *GraphConnector) userToken(accountID string) (string, error) {
	g.mu.RLock()
	session := g.sessions[accountID]
	if session != nil && session.UserToken != "" {
		g.mu.RUnlock()
		return session.UserToken, nil
	}
	g.mu.RUnlock()

	if g.db != nil {
		var model database.MetaConnectionModel
		if err := g.db.First(&model, "account_id = ?", accountID).Error; err == nil {
			g.mu.Lock()
			if g.sessions[accountID] == nil {
				g.sessions[accountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[accountID].UserToken = model.UserToken
			g.mu.Unlock()
			return model.UserToken, nil
		}
	}
	return "", errors.New("facebook connection not found")
}"""
content = re.sub(user_token_re, user_token_new, content, flags=re.DOTALL)

find_page_re = r'func \(g \*GraphConnector\) findPage\(pageID string\) \(string, graphPage, bool\) \{.*?\n\}'
find_page_new = """func (g *GraphConnector) findPage(pageID string) (string, graphPage, bool) {
	g.mu.RLock()
	for accountID, session := range g.sessions {
		if page, ok := session.Pages[pageID]; ok {
			g.mu.RUnlock()
			return accountID, page, true
		}
	}
	g.mu.RUnlock()

	if g.db != nil {
		var model database.MetaPageTokenModel
		if err := g.db.First(&model, "page_id = ?", pageID).Error; err == nil {
			page := graphPage{
				MetaPage: domain.MetaPage{
					ID:         model.PageID,
					Name:       model.Name,
					Category:   model.Category,
					PictureURL: model.PictureURL,
					Connected:  model.IsConnected,
					TokenReady: model.AccessToken != "",
				},
				AccessToken: model.AccessToken,
			}
			g.mu.Lock()
			if g.sessions[model.AccountID] == nil {
				g.sessions[model.AccountID] = &graphSession{Pages: make(map[string]graphPage)}
			}
			g.sessions[model.AccountID].Pages[model.PageID] = page
			g.mu.Unlock()
			return model.AccountID, page, true
		}
	}
	return "", graphPage{}, false
}"""
content = re.sub(find_page_re, find_page_new, content, flags=re.DOTALL)

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
