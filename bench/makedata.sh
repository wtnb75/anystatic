#! /bin/sh
# make test data in current directory
set -eu

python3 - <<'PY'
open('small.txt','wb').write(b'a'*4096+b'\n')
open('medium.txt','wb').write(b'a'*100000+b'\n')
open('large.txt','wb').write(b'a'*1048576+b'\n')
PY

/compr-recursive compress -dir .
