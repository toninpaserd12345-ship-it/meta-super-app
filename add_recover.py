import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("""func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) (items []domain.MetaAdAccount, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ListAdAccounts PANIC", "panic", r)
			items = []domain.MetaAdAccount{}
			err = nil
		}
	}()
	token, err := g.userToken(accountID)""", """func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) (items []domain.MetaAdAccount, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ListAdAccounts PANIC", "panic", r)
			items = []domain.MetaAdAccount{}
			err = nil
		}
	}()
	token, err2 := g.userToken(accountID)
	if err2 != nil {
		slog.Error("ListAdAccounts userToken failed", "error", err2)
		return []domain.MetaAdAccount{}, nil
	}""")

content = content.replace("if err != nil {\\n\\t\\tslog.Error(\\"ListAdAccounts userToken failed\\", \\"error\\", err)\\n\\t\\treturn []domain.MetaAdAccount{}, nil\\n\\t}", "")

# Also need to fix getJSON err
content = content.replace("if err = g.getJSON(ctx, \\"https://graph.facebook.com/\\"+g.cfg.Version+\\"/me/adaccounts?\\"+query.Encode(), &response); err != nil {", "if err2 := g.getJSON(ctx, \\"https://graph.facebook.com/\\"+g.cfg.Version+\\"/me/adaccounts?\\"+query.Encode(), &response); err2 != nil {")
content = content.replace("slog.Error(\\"ListAdAccounts getJSON failed\\", \\"error\\", err)", "slog.Error(\\"ListAdAccounts getJSON failed\\", \\"error\\", err2)")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
