package hammock

import (
	"fmt"

	"github.com/mikebell-org/go-couchdb"
)

type CouchDB struct {
	couchdb.CouchDB
}

func Database(host, database, username, password string) (*CouchDB, error) {

	db, err := couchdb.Database(host, database, username, password)

	return &CouchDB{*db}, err
}

func Sync(db *CouchDB, path string) (changes []string, err error) {

	// TODO: implement a document freezing option

	disk_data := newDesignDocCollection()

	if err = disk_data.loadFromDisk(path); err == nil {

		db_data := newDesignDocument()

		for doc_name, document := range disk_data.Documents {

			if err = db.GetDocument(&db_data, fmt.Sprintf("%v", doc_name)); err == nil {

				if updated, doc_changes := db_data.update(document); updated {

					changes = append(changes, doc_changes...)

					if success, e := db.PutDocument(db_data, doc_name); e != nil || !success.OK {

						err = e
						return
					}
				}
			}
		}
	}

	return
}
