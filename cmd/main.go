package main

import (
	"fmt"
	"log"

	"github.com/mycok/uSearch/internal/textindexer/store/memory"
)

func main() {
	idx, err := memory.NewInMemoryBleveIndexer()

	if err != nil {
		log.Println(fmt.Errorf("----newInMemoryBleveIndexer error-----: %w", err))
	}

	log.Println(fmt.Printf("----This is the new indexer instance----: %+#v", idx))

	if err = idx.Close(); err != nil {
		log.Println(fmt.Printf("----closing indexer instance----: %+#v", err))
	}
}
