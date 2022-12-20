package main

import (
	"sort"
	"strings"
)

type Word struct {
	Word   string
	PosId  uint32
	PageId uint32
	LineId uint32
}

type Page struct {
	Words     []*Word
	WordMap   map[string][]int
	WordMapV2 map[string][]*Word
}

type Supplier struct {
	SupplierName string
	Id           string
}

type SuppliersForPage struct {
	Page      *Page
	Suppliers []*Supplier
}

// SearchSupplierFromPage - find supplier name from a page
// return nil if the supplier name is not found
func SearchSupplierFromPage(pages []*Page, supplier *Supplier) *Supplier {
	for _, page := range pages {
		canMatch := matchSupplierNameInPage(strings.Split(supplier.SupplierName, " "), page)
		if canMatch {
			return supplier
		}
	}
	return nil
}

// SearchSupplierFromPageV2 - find supplier name from a page
// return nil if the supplier name is not found
func SearchSupplierFromPageV2(pages []*Page, supplier *Supplier) *Supplier {
	for _, page := range pages {
		canMatch := matchSupplierNameInPageV2(strings.Split(supplier.SupplierName, " "), page)
		if canMatch {
			return supplier
		}
	}
	return nil
}

// SearchSupplierFromPageV2 - find supplier name from a page
// return nil if the supplier name is not found
func SearchSupplierFromPageV3(potentialSuppliersForPage []*SuppliersForPage) (supplier *Supplier) {
	for _, suppliersForPage := range potentialSuppliersForPage {
		for _, supplier := range suppliersForPage.Suppliers {
			page := suppliersForPage.Page
			canMatch := matchSupplierNameInPageV3(strings.Split(supplier.SupplierName, " "), page, nil)
			if canMatch {
				return supplier
			}
		}
	}
	return nil
}

// groupInvoiceWords - group words in invoice file by page id
func groupInvoiceWords(words []*Word) (pages []*Page) {
	pages = make([]*Page, 0)
	pageMap := make(map[uint32]*Page)
	for _, word := range words {
		page, ok := pageMap[word.PageId]
		if !ok {
			page = &Page{Words: make([]*Word, 0, 1)}
			pageMap[word.PageId] = page
			pages = append(pages, page)
		}
		page.Words = append(page.Words, word)
	}
	return pages
}

// matchSupplierNameInPage - match supplier name in the page
func matchSupplierNameInPage(supplierNameToken []string, page *Page) (canMatch bool) {
	if page == nil {
		return false
	}
	lenName := len(supplierNameToken)
	lenPage := len(page.Words)
	if lenName == 0 || lenPage == 0 {
		return false
	}
	idxName := 0
	idxWord := 0
	for idxName < lenName && idxWord < lenPage {
		if supplierNameToken[idxName] == page.Words[idxWord].Word {
			idxName++
		}
		idxWord++
	}
	return idxName == lenName
}

// matchSupplierNameInPageV2 - match supplier name in the page
func matchSupplierNameInPageV2(supplierNameToken []string, page *Page) (canMatch bool) {
	if page == nil {
		return false
	}
	if len(supplierNameToken) == 0 || len(page.WordMap) == 0 {
		return
	}
	idxWord := -1
	for _, token := range supplierNameToken {
		wordList, ok := page.WordMap[token]
		if !ok {
			return false
		}
		res := sort.SearchInts(wordList, idxWord+1) // use binary search to find the next idx
		if res == len(wordList) {                   // not found
			return false
		}
		idxWord = wordList[res] // jump to the next idx
	}
	return true
}

// matchSupplierNameInPageV3 - match supplier name in the page
func matchSupplierNameInPageV3(supplierNameToken []string, page *Page, startWord *Word) (canMatch bool) {
	if page == nil {
		return false
	}
	if len(supplierNameToken) == 0 {
		return true
	}
	if len(page.WordMapV2) == 0 {
		return false
	}

	token := supplierNameToken[0]
	wordList, ok := page.WordMapV2[token]
	if !ok {
		return false
	}
	// use binary search to find the next idx
	res := sort.Search(len(wordList), func(i int) bool {
		wi := wordList[i]
		wj := startWord
		return startWord == nil || wi.LineId > wj.LineId || wi.LineId == wj.LineId && wi.PosId > wj.PosId
	})
	if res == len(wordList) { // not found
		return false
	}

	for i := res; i < len(wordList); i++ {
		nextStartWord := wordList[i]
		if startWord == nil || startWord.LineId+1 >= nextStartWord.LineId {
			if matchSupplierNameInPageV3(supplierNameToken[1:], page, nextStartWord) {
				return true
			}
		}
	}
	return false
}

// sortWordsInPage - sort the words by position id and line id
func sortWordsInPage(page *Page) *Page {
	if page == nil || len(page.Words) == 0 {
		return page
	}
	sort.Sort(byPosAndLine(page.Words))
	return page
}

func buildWordMapInPage(page *Page) {
	if page == nil {
		return
	}
	page.WordMap = make(map[string][]int)
	for idx, w := range page.Words {
		wordList, ok := page.WordMap[w.Word]
		if !ok {
			wordList = make([]int, 0)
		}
		wordList = append(wordList, idx)
		page.WordMap[w.Word] = wordList
	}
}

func buildWordMapV2InPage(page *Page) {
	if page == nil {
		return
	}
	page.WordMapV2 = make(map[string][]*Word)
	for _, w := range page.Words {
		wordList, ok := page.WordMapV2[w.Word]
		if !ok {
			wordList = make([]*Word, 0)
		}
		wordList = append(wordList, w)
		page.WordMapV2[w.Word] = wordList
	}
}

// byPosAndLine - sort words by position id and line id
type byPosAndLine []*Word

func (s byPosAndLine) Len() int {
	return len(s)
}

func (s byPosAndLine) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPosAndLine) Less(i, j int) bool {
	wi := s[i]
	wj := s[j]
	return wi.LineId < wj.LineId || (wi.LineId == wj.LineId && wi.PosId < wj.PosId)
}
