package hammock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type designDocCollection struct {
	Documents map[string]*designDocument
}

func newDesignDocCollection() *designDocCollection {

	return &designDocCollection{Documents: map[string]*designDocument{}}
}

func (data *designDocCollection) loadFromDisk(designs_root string) error {

	// read all design docs saved at designs_root
	err := filepath.Walk(designs_root, func(path string, f os.FileInfo, err error) error {

		if path == designs_root {

			return nil
		}

		if f.IsDir() {

			parts := strings.Split(path, string(filepath.Separator))
			num_parts := len(parts)
			doc_name := fmt.Sprintf("_design/%v", parts[num_parts-1])

			if _, ok := data.Documents[doc_name]; !ok {

				data.Documents[doc_name] = newDesignDocument(doc_name)
			}

			data.Documents[doc_name].loadFromDisk(path)

			return filepath.SkipDir
		}

		return nil
	})

	return err
}
