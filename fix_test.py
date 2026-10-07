import re
content = open("backend/internal/infrastructure/meta/graph_connector_test.go").read()

content = re.sub(r'NewGraphConnector\((.*?)\)', r'NewGraphConnector(\1, nil)', content)

open("backend/internal/infrastructure/meta/graph_connector_test.go", "w").write(content)
