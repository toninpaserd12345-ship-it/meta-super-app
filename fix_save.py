import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("g.sessions[stateData.AccountID] = session", """g.sessions[stateData.AccountID] = session
	if g.db != nil {
		g.db.Save(&database.MetaConnectionModel{AccountID: stateData.AccountID, UserToken: token.AccessToken})
	}""")

content = content.replace("session.Pages[pageID] = page\n\t_ = g.persistStateLocked()", """session.Pages[pageID] = page
	if g.db != nil {
		g.db.Save(&database.MetaPageTokenModel{
			PageID:      pageID,
			AccountID:   accountID,
			Name:        page.Name,
			Category:    page.Category,
			PictureURL:  page.PictureURL,
			AccessToken: page.AccessToken,
			IsConnected: page.Connected,
		})
	}
	_ = g.persistStateLocked()""")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
