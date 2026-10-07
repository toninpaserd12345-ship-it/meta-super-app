import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("""func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return []domain.MetaAdAccount{}, nil
	}""", """func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		slog.Error("ListAdAccounts userToken failed", "error", err)
		return []domain.MetaAdAccount{}, nil
	}""")

content = content.replace("""	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		return nil, err
	}""", """	if err = g.getJSON(ctx, "https://graph.facebook.com/"+g.cfg.Version+"/me/adaccounts?"+query.Encode(), &response); err != nil {
		slog.Error("ListAdAccounts getJSON failed", "error", err)
		return []domain.MetaAdAccount{}, nil
	}""")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
