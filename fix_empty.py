content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

content = content.replace("""func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}""", """func (g *GraphConnector) ListAdAccounts(ctx context.Context, _ string, accountID string) ([]domain.MetaAdAccount, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return []domain.MetaAdAccount{}, nil
	}""")

content = content.replace("""func (g *GraphConnector) ListCampaigns(ctx context.Context, _ string, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}""", """func (g *GraphConnector) ListCampaigns(ctx context.Context, _ string, accountID, adAccountID string) ([]domain.MetaCampaign, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return []domain.MetaCampaign{}, nil
	}""")

content = content.replace("""func (g *GraphConnector) ListAds(ctx context.Context, _ string, accountID, campaignID string) ([]domain.MetaAd, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return nil, err
	}""", """func (g *GraphConnector) ListAds(ctx context.Context, _ string, accountID, campaignID string) ([]domain.MetaAd, error) {
	token, err := g.userToken(accountID)
	if err != nil {
		return []domain.MetaAd{}, nil
	}""")

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
