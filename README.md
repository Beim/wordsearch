# wordsearch

# Prerequisite

- install golang https://go.dev/doc/install

# Run

```bash
go run ./solution -invoice=invoice.txt -supplier=suppliernames.txt -worker=5
# expected result
# supplier name found: 3153303,Demo Company
```

# Requirement

- find the supplier name of the invoice by matching the given list of supplier names to the invoice.
- the solution should be scalable to hundreds of thousands of supplier names.

# Assumptions

1. There is only one match supplier name for given input.
2. The words of a supplier name are on the same page.
3. The words of a supplier name may not be in the same line.
4. The sequence of word concatenation is from left to right, e.g. "word=Company, pos_id=0" and "word=Demo, pos_id=1" cannot match "Demo Company", but can match "Company Demo".
5. The words in invoice.txt can only be concatenated by space to match the supplier name.
6. The supplier name is an exact match, e.g. "Demo.Company" can't match "Demo-Company".
7. The word size in an invoice is limited, in another word the scalable requirement is only for suppliernames.txt.

# Solution

1. Preprocess the words in invoice.txt, group them by page id.
2. Preprocess the suppliername.txt, create a channel to load the supplier name, prevent loading all supplier names into memory.
3. Start worker to match the words in invoice with the supplier names. For this step I provided **two implementations**
   1. solution1 - [matchSupplierNameInPage](https://github.com/Beim/wordsearch/blob/de8331f17c3596ac8ac0d058ab1c56762e3ee8a5/solution/search.go#L66) - use two pointer to scan the words in both supplier name and invoice file.
   2. solution2 - [matchSupplierNameInPageV2](https://github.com/Beim/wordsearch/blob/de8331f17c3596ac8ac0d058ab1c56762e3ee8a5/solution/search.go#L87) - use binary search to optimize the scan of words in invoice file.
4. If one of the worker can find the supplier name, stop all other workers.
5. Print out the supplier name.

## Time complexity

- for solution1 - `O(m * n)` where `m` is the number of words in an invoice, and `n` is the number of supplier names.
- for solution2 -  `O(m * p)` where `p` is the number of pages in an invoice, and `m` is the number of supplier names.

## Space complexity

- `O(m)` where `m` is the number of words in an invoice.
