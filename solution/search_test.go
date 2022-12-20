package main

import (
	"reflect"
	"testing"
)

func Search(invoiceWords []*Word, suppliers []*Supplier) (supplier *Supplier) {
	pages := groupInvoiceWords(invoiceWords)
	for _, page := range pages {
		sortWordsInPage(page)
	}
	for _, s := range suppliers {
		supplier = SearchSupplierFromPage(pages, s)
		if supplier != nil {
			return
		}
	}
	return nil
}

func TestSearch(t *testing.T) {
	type args struct {
		invoiceWords []*Word
		suppliers    []*Supplier
	}
	tests := []struct {
		name         string
		args         args
		wantSupplier *Supplier
	}{
		{
			name: "given sample",
			args: args{
				invoiceWords: []*Word{
					{Word: "INVOICE", PageId: 1, LineId: 0, PosId: 1},
					{Word: "Demo", PageId: 1, LineId: 4, PosId: 0},
					{Word: "Company", PageId: 1, LineId: 4, PosId: 1},
				},
				suppliers: []*Supplier{
					{SupplierName: "Demo Company", Id: "123"},
				},
			},
			wantSupplier: &Supplier{
				SupplierName: "Demo Company",
				Id:           "123",
			},
		},
		{
			name: "Words in different page",
			args: args{
				invoiceWords: []*Word{
					{Word: "INVOICE", PageId: 1, LineId: 0, PosId: 1},
					{Word: "Demo", PageId: 2, LineId: 4, PosId: 0},
					{Word: "Company", PageId: 1, LineId: 4, PosId: 1},
				},
				suppliers: []*Supplier{
					{SupplierName: "Demo Company", Id: "123"},
				},
			},
			wantSupplier: nil,
		},
		{
			name: "Words in different line",
			args: args{
				invoiceWords: []*Word{
					{Word: "INVOICE", PageId: 1, LineId: 0, PosId: 1},
					{Word: "Demo", PageId: 1, LineId: 3, PosId: 2},
					{Word: "invoice", PageId: 1, LineId: 3, PosId: 3},
					{Word: "Company", PageId: 1, LineId: 4, PosId: 1},
				},
				suppliers: []*Supplier{
					{SupplierName: "Demo Company", Id: "123"},
				},
			},
			wantSupplier: &Supplier{
				SupplierName: "Demo Company",
				Id:           "123",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSupplier := Search(tt.args.invoiceWords, tt.args.suppliers); !reflect.DeepEqual(gotSupplier, tt.wantSupplier) {
				t.Errorf("Search() = %v, want %v", gotSupplier, tt.wantSupplier)
			}
		})
	}
}

func Test_groupInvoiceWords(t *testing.T) {
	type args struct {
		words []*Word
	}
	tests := []struct {
		name      string
		args      args
		wantPages []*Page
	}{
		{
			name: "two groups in different line",
			args: args{
				words: []*Word{
					{Word: "word1", PosId: 0, PageId: 0, LineId: 0},
					{Word: "word2", PosId: 0, PageId: 0, LineId: 1},
					{Word: "word3", PosId: 0, PageId: 0, LineId: 0},
				},
			},
			wantPages: []*Page{
				{
					Words: []*Word{
						{Word: "word1", PosId: 0, PageId: 0, LineId: 0},
						{Word: "word2", PosId: 0, PageId: 0, LineId: 1},
						{Word: "word3", PosId: 0, PageId: 0, LineId: 0},
					},
				},
			},
		},
		{
			name: "two groups in different page",
			args: args{
				words: []*Word{
					{Word: "word1", PosId: 0, PageId: 0, LineId: 0},
					{Word: "word2", PosId: 0, PageId: 1, LineId: 0},
					{Word: "word3", PosId: 0, PageId: 0, LineId: 0},
				},
			},
			wantPages: []*Page{
				{
					Words: []*Word{
						{Word: "word1", PosId: 0, PageId: 0, LineId: 0},
						{Word: "word3", PosId: 0, PageId: 0, LineId: 0},
					},
				},
				{
					Words: []*Word{
						{Word: "word2", PosId: 0, PageId: 1, LineId: 0},
					},
				},
			},
		},
		{
			name: "empty words",
			args: args{
				words: []*Word{},
			},
			wantPages: []*Page{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotGroupedWords := groupInvoiceWords(tt.args.words); !reflect.DeepEqual(gotGroupedWords, tt.wantPages) {
				t.Errorf("groupInvoiceWords() = %v, want %v", gotGroupedWords, tt.wantPages)
			}
		})
	}
}

func Test_searchSupplierNameInPage(t *testing.T) {
	type args struct {
		supplierNameToken []string
		page              *Page
	}
	tests := []struct {
		name         string
		args         args
		wantCanMatch bool
	}{
		{
			name: "match in same line",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					Words: []*Word{
						{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
						{Word: "Company", PosId: 1, LineId: 0, PageId: 0},
					},
				},
			},
			wantCanMatch: true,
		},
		{
			name: "match in different line",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					Words: []*Word{
						{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
						{Word: "Company", PosId: 0, LineId: 1, PageId: 0},
					},
				},
			},
			wantCanMatch: true,
		},
		{
			name: "not match",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					Words: []*Word{
						{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
						{Word: "AnotherCompany", PosId: 0, LineId: 1, PageId: 0},
					},
				},
			},
			wantCanMatch: false,
		},
		{
			name: "empty supplier name",
			args: args{
				supplierNameToken: []string{},
				page: &Page{
					Words: []*Word{
						{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
						{Word: "AnotherCompany", PosId: 0, LineId: 1, PageId: 0},
					},
				},
			},
			wantCanMatch: false,
		},
		{
			name: "empty page",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					Words: []*Word{},
				},
			},
			wantCanMatch: false,
		},
		{
			name: "invalid page",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page:              nil,
			},
			wantCanMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotCanMatch := matchSupplierNameInPage(tt.args.supplierNameToken, tt.args.page); !reflect.DeepEqual(gotCanMatch, tt.wantCanMatch) {
				t.Errorf("matchSupplierNameInPage() = %v, want %v", gotCanMatch, tt.wantCanMatch)
			}
		})
	}
}

func Test_sortWordsInPage(t *testing.T) {
	type args struct {
		page *Page
	}
	tests := []struct {
		name string
		args args
		want *Page
	}{
		{
			name: "words in different line",
			args: args{
				page: &Page{
					Words: []*Word{
						{Word: "Company", PosId: 1, LineId: 1, PageId: 0},
						{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
						{Word: "INVOICE", PosId: 0, LineId: 1, PageId: 0},
					},
				},
			},
			want: &Page{
				Words: []*Word{
					{Word: "Demo", PosId: 0, LineId: 0, PageId: 0},
					{Word: "INVOICE", PosId: 0, LineId: 1, PageId: 0},
					{Word: "Company", PosId: 1, LineId: 1, PageId: 0},
				},
			},
		},
		{
			name: "empty page",
			args: args{
				page: &Page{
					Words: []*Word{},
				},
			},
			want: &Page{
				Words: []*Word{},
			},
		},
		{
			name: "invalid page",
			args: args{
				page: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sortWordsInPage(tt.args.page); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortWordsInPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_matchSupplierNameInPageV2(t *testing.T) {
	type args struct {
		supplierNameToken []string
		page              *Page
	}
	tests := []struct {
		name         string
		args         args
		wantCanMatch bool
	}{
		{
			name: "exact match",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					WordMap: map[string][]int{
						"Demo":    {1, 3},
						"Company": {2},
					},
				},
			},
			wantCanMatch: true,
		},
		{
			name: "not match",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page: &Page{
					WordMap: map[string][]int{
						"Demo":    {3, 4},
						"Company": {2},
					},
				},
			},
			wantCanMatch: false,
		},
		{
			name: "invalid page",
			args: args{
				supplierNameToken: []string{"Demo", "Company"},
				page:              nil,
			},
			wantCanMatch: false,
		},
		{
			name: "empty supplier name",
			args: args{
				supplierNameToken: []string{},
				page: &Page{
					WordMap: map[string][]int{
						"Demo":    {1, 3},
						"Company": {2},
					},
				},
			},
			wantCanMatch: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotCanMatch := matchSupplierNameInPageV2(tt.args.supplierNameToken, tt.args.page); gotCanMatch != tt.wantCanMatch {
				t.Errorf("matchSupplierNameInPageV2() = %v, want %v", gotCanMatch, tt.wantCanMatch)
			}
		})
	}
}

func Test_matchSupplierNameInPageV3(t *testing.T) {
	type args struct {
		supplierNameToken []string
		page              *Page
	}
	tests := []struct {
		name         string
		args         args
		wantCanMatch bool
	}{
		// TODO: Add test cases.
		{
			name: "not match",
			args: args{
				supplierNameToken: []string{
					"Demo", "Company",
				},
				page: &Page{
					WordMapV2: map[string][]*Word{
						"Demo": {
							{
								LineId: 0,
							},
						},
						"Company": {
							{
								LineId: 20,
							},
						},
					},
				},
			},
			wantCanMatch: false,
		},
		{
			name: "can match",
			args: args{
				supplierNameToken: []string{
					"Demo", "Company",
				},
				page: &Page{
					WordMapV2: map[string][]*Word{
						"Demo": {
							{
								LineId: 0,
							},
							{
								LineId: 19,
							},
						},
						"Company": {
							{
								LineId: 20,
							},
						},
					},
				},
			},
			wantCanMatch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotCanMatch := matchSupplierNameInPageV3(tt.args.supplierNameToken, tt.args.page, nil); gotCanMatch != tt.wantCanMatch {
				t.Errorf("matchSupplierNameInPageV3() = %v, want %v", gotCanMatch, tt.wantCanMatch)
			}
		})
	}
}
