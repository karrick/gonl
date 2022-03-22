.PHONY: bench clean distclean test

bench: 2600-0.txt
	go test -bench=.

clean:
	rm -rf logs

distclean: clean
	rm -f 2600-0.txt

test: 2600-0.txt
	go test

2600-0.txt:
	curl -LOC - https://gutenberg.org/files/2600/2600-0.txt
