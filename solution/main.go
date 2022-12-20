package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	CMD_SEARCH    = "search"
	CMD_INDEX     = "index"
	CMD_SEARCH_V2 = "searchv2"
)

func main() {
	invoiceFilePath := flag.String("invoice", "invoice.txt", "words of an invoice")
	supplierNameFilePath := flag.String("supplier", "suppliernames.txt", "a list of supplier names")
	cmd := flag.String("cmd", CMD_SEARCH, "run command search,index")
	workerNum := flag.Uint64("worker", 5, "number of workers")
	flag.Parse()

	if *cmd == CMD_SEARCH {
		if err := FindSupplierName(*invoiceFilePath, *supplierNameFilePath, *workerNum); err != nil {
			log.Fatal(err)
		}
	} else if *cmd == CMD_INDEX {
		if err := BuildIndex(*supplierNameFilePath); err != nil {
			log.Fatal(err)
		}
	} else if *cmd == CMD_SEARCH_V2 {
		if err := FindSupplierNameV2(*invoiceFilePath, *supplierNameFilePath, *workerNum); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("invalid cmd")
	}

}

func BuildIndex(supplierNameFilePath string) (err error) {
	supplierChan, err := loadSupplierNameFile(supplierNameFilePath)
	if err != nil {
		return err
	}

	supplierMap := map[string][]*Supplier{}
	for supplier := range supplierChan {
		nameToken := strings.Split(supplier.SupplierName, " ")
		if len(nameToken) < 1 {
			log.Fatal("invalid supplier name")
		}
		firstName := nameToken[0]
		suppliers, ok := supplierMap[firstName]
		if !ok {
			suppliers = make([]*Supplier, 0)
		}
		suppliers = append(suppliers, supplier)
		supplierMap[firstName] = suppliers
	}

	indexMap := map[string]uint64{}
	f, err := os.Create(fmt.Sprintf("%s.indexed", supplierNameFilePath))
	defer f.Close()
	if err != nil {
		return err
	}
	currentIdx := uint64(0)
	for firstName, suppliers := range supplierMap {
		var buf bytes.Buffer
		for _, supplier := range suppliers {
			_, err := buf.WriteString(fmt.Sprintf("%s,%s\n", supplier.Id, supplier.SupplierName))
			if err != nil {
				return err
			}
		}
		n, err := f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		indexMap[firstName] = currentIdx
		currentIdx += uint64(n)
	}
	idxf, err := os.Create(fmt.Sprintf("%s.idx", supplierNameFilePath))
	defer idxf.Close()
	if err != nil {
		return err
	}
	indexJson, err := json.Marshal(indexMap)
	if err != nil {
		return err
	}
	_, err = idxf.Write(indexJson)
	return
}

func FindSupplierNameV2(invoiceFilePath, supplierNameFilePath string, workerNum uint64) (err error) {
	if workerNum == 0 {
		return fmt.Errorf("invalid worker num")
	}

	// preprocess the invoice file
	words, err := loadInvoiceFile(invoiceFilePath)
	if err != nil {
		return err
	}
	pages := groupInvoiceWords(words)
	for _, page := range pages {
		sortWordsInPage(page)
		buildWordMapInPage(page)
	}

	indexMap, supplierNameFile, err := loadSupplierNameFileWithIndex(supplierNameFilePath)
	if err != nil {
		return
	}
	potentialSuppliersForPage, err := filterPotentialSuppliersForPage(pages, indexMap, supplierNameFile)
	if err != nil {
		return err
	}

	supplier := SearchSupplierFromPageV3(potentialSuppliersForPage)
	if supplier != nil {
		log.Printf("supplier name found: %s,%s", supplier.Id, supplier.SupplierName)
	} else {
		log.Println("supplier name not found")
	}

	return nil
}

func filterPotentialSuppliersForPage(pages []*Page, indexMap map[string]uint64, supplierNameFile *os.File) (suppliersForPage []*SuppliersForPage, err error) {
	suppliersForPage = make([]*SuppliersForPage, 0)
	for _, page := range pages {
		suppliers := make([]*Supplier, 0)
		for _, word := range page.Words {
			idx, ok := indexMap[word.Word]
			if !ok {
				continue
			}
			_, err = supplierNameFile.Seek(int64(idx), 0)
			if err != nil {
				return
			}
			reg := regexp.MustCompile(`(\d+),(.+)`)
			scanner := bufio.NewScanner(supplierNameFile)
			for scanner.Scan() {
				line := scanner.Text()
				match := reg.FindStringSubmatch(line)
				if len(match) != 3 {
					err = fmt.Errorf("invalid supplier name text")
					return
				}
				id := match[1]
				supplierName := match[2]
				suppliers = append(suppliers, &Supplier{
					Id:           id,
					SupplierName: supplierName,
				})
				break
			}
		}
		if len(suppliers) > 0 {
			suppliersForPage = append(suppliersForPage, &SuppliersForPage{
				Page:      page,
				Suppliers: suppliers,
			})
		}
	}
	return
}

func loadSupplierNameFileWithIndex(supplierNameFilePath string) (indexMap map[string]uint64, supplierNameFile *os.File, err error) {
	idxf, err := os.Open(fmt.Sprintf("%s.idx", supplierNameFilePath))
	if err != nil {
		return
	}
	idxJson, err := ioutil.ReadAll(idxf)
	if err != nil {
		return
	}
	err = json.Unmarshal(idxJson, &indexMap)
	if err != nil {
		return
	}
	supplierNameFile, err = os.Open(fmt.Sprintf("%s.indexed", supplierNameFilePath))
	return
}

// FindSupplierName - find the supplier name from input files
// if supplier name is found, it will print the id and supplier name of the supplier
func FindSupplierName(invoiceFilePath, supplierNameFilePath string, workerNum uint64) (err error) {
	if workerNum == 0 {
		return fmt.Errorf("invalid worker num")
	}

	// preprocess the invoice file
	words, err := loadInvoiceFile(invoiceFilePath)
	if err != nil {
		return err
	}
	pages := groupInvoiceWords(words)
	for _, page := range pages {
		sortWordsInPage(page)
		buildWordMapInPage(page)
	}

	// preprocess the supplier name file
	supplierChan, err := loadSupplierNameFile(supplierNameFilePath)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	done := make(chan bool, 1)
	// send worker job
	for i := uint64(0); i < workerNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWorker(pages, supplierChan, done)
		}()
	}
	// wait for all worker complete
	wg.Wait()
	select {
	case <-done:
		return nil
	default:
		log.Println("supplier name not found")
	}
	return nil
}

// runWorker - run worker to find the supplier name
func runWorker(pages []*Page, supplierChan chan *Supplier, done chan bool) {
	for supplier := range supplierChan {
		select {
		case <-done: // stop early if other worker has found the supplier name
			done <- true
			return
		default:
			s := SearchSupplierFromPageV2(pages, supplier)
			if s != nil {
				log.Printf("supplier name found: %s,%s", s.Id, s.SupplierName)
				done <- true
				return
			}
		}
	}
}

// loadSupplierNameFile - load supplier names from file asynchronously
func loadSupplierNameFile(supplierNameFilePath string) (supplierChan chan *Supplier, err error) {
	bufSize := 100
	supplierChan = make(chan *Supplier, bufSize)
	supplierNameFile, err := os.Open(supplierNameFilePath)
	if err != nil {
		return nil, err
	}
	go func() {
		reg := regexp.MustCompile(`(\d+),(.+)`)
		defer supplierNameFile.Close()
		scanner := bufio.NewScanner(supplierNameFile)
		scanner.Scan() // skip the first line
		for scanner.Scan() {
			line := scanner.Text()
			match := reg.FindStringSubmatch(line)
			if len(match) != 3 {
				log.Printf("invalid supplier name text: %s", line)
				return
			}
			id := match[1]
			supplierName := match[2]
			supplierChan <- &Supplier{
				Id:           id,
				SupplierName: supplierName,
			}
		}
		close(supplierChan)
	}()
	return
}

// loadInvoiceFile - load words of an invoice from file
func loadInvoiceFile(invoiceFilePath string) (words []*Word, err error) {
	// use regexp instead of json package because the file content is not valid JSON
	reg := regexp.MustCompile(`'pos_id': (\d+), .+'word': '(.+)', 'line_id': (\d+), .+'page_id': (\d+),`)
	invoiceFile, err := os.Open(invoiceFilePath)
	if err != nil {
		return
	}
	defer invoiceFile.Close()
	words = make([]*Word, 0)
	scanner := bufio.NewScanner(invoiceFile)
	for scanner.Scan() {
		line := scanner.Text()
		match := reg.FindStringSubmatch(line)
		if len(match) != 5 {
			err = fmt.Errorf("invalid invoice text: %s", line)
			return
		}
		posIdStr := match[1]
		word := match[2]
		lineIdStr := match[3]
		pageIdStr := match[4]
		posId, err := strconv.ParseUint(posIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		lineId, err := strconv.ParseUint(lineIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		pageId, err := strconv.ParseUint(pageIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		words = append(words, &Word{
			Word:   word,
			PosId:  uint32(posId),
			LineId: uint32(lineId),
			PageId: uint32(pageId),
		})
	}
	return
}
