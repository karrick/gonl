.PHONY: bench clean

bench: 2600-h.htm
	go test -bench=.

clean:
	rm -f 2600-h.htm

2600-h.htm:
	curl -LOC - https://www.gutenberg.org/files/2600/2600-h/2600-h.htm
