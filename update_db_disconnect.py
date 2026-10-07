import re

content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

# Update DisconnectPage
# Inside DisconnectPage, find:
# 	page.Connected = false
#	session.Pages[pageID] = page
#   _ = g.persistStateLocked()
new_disconnect = """	page.Connected = false
	session.Pages[pageID] = page
	if g.db != nil {
		g.db.Model(&database.MetaPageTokenModel{}).Where("page_id = ?", pageID).Update("is_connected", false)
	}
	_ = g.persistStateLocked()"""
content = content.replace("page.Connected = false\n\tsession.Pages[pageID] = page\n\t_ = g.persistStateLocked()", new_disconnect)

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
