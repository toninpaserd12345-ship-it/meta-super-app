import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("""	if err != nil {
		slog.Error("ListAdAccounts userToken failed", "error", err)
		return []domain.MetaAdAccount{}, nil
	}""", """	if err != nil {
		return nil, fmt.Errorf("facebook connection not found or invalid: %w", err)
	}""")

content = content.replace("""	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		slog.Error("ListAdAccounts getJSON failed", "error", err)
		return []domain.MetaAdAccount{}, nil
	}""", """	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		return nil, fmt.Errorf("facebook api error: %w", err)
	}""")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
