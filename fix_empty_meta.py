import re
content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

# Replace ListAdAccounts
content = re.sub(r'token, err := g\.userToken\(accountID\)\n\tif err != nil \{\n\t\treturn nil, err\n\t\}',
                 'token, err := g.userToken(accountID)\n\tif err != nil {\n\t\treturn []domain.MetaAdAccount{}, nil\n\t}', content)

# Replace ListCampaigns
content = re.sub(r'func \(g \*GraphConnector\) ListCampaigns\(ctx context.Context, _ string, accountID, adAccountID string\) \(\[\]domain\.MetaCampaign, error\) \{\n\ttoken, err := g\.userToken\(accountID\)\n\tif err != nil \{\n\t\treturn nil, err\n\t\}',
                 'func (g *GraphConnector) ListCampaigns(ctx context.Context, _ string, accountID, adAccountID string) ([]domain.MetaCampaign, error) {\n\ttoken, err := g.userToken(accountID)\n\tif err != nil {\n\t\treturn []domain.MetaCampaign{}, nil\n\t}', content)

# Replace ListAds
content = re.sub(r'func \(g \*GraphConnector\) ListAds\(ctx context.Context, _ string, accountID, campaignID string\) \(\[\]domain\.MetaAd, error\) \{\n\ttoken, err := g\.userToken\(accountID\)\n\tif err != nil \{\n\t\treturn nil, err\n\t\}',
                 'func (g *GraphConnector) ListAds(ctx context.Context, _ string, accountID, campaignID string) ([]domain.MetaAd, error) {\n\ttoken, err := g.userToken(accountID)\n\tif err != nil {\n\t\treturn []domain.MetaAd{}, nil\n\t}', content)

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
