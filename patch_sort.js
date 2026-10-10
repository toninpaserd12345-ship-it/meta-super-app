const fs = require('fs');
let file = fs.readFileSync('backend/internal/infrastructure/meta/graph_connector.go', 'utf8');

file = file.replace(
  /result := make\(\[\]domain\.MetaPage, 0, len\(pages\)\)\n\tfor _, page := range pages \{\n\t\tresult = append\(result, page\.MetaPage\)\n\t\}\n\treturn result, nil/g,
  `result := make([]domain.MetaPage, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.MetaPage)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].ID < result[j].ID
		}
		return result[i].Name < result[j].Name
	})
	return result, nil`
);

if (!file.includes('"sort"')) {
  file = file.replace(/"strings"\n/g, '"strings"\n\t"sort"\n');
}

fs.writeFileSync('backend/internal/infrastructure/meta/graph_connector.go', file);
