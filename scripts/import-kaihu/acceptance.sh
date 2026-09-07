#!/bin/sh
# 机械验收 A/B/C(D push 由收尾流程执行)。用法: acceptance.sh <xlsx路径> [outdir]
set -u
if [ $# -lt 1 ]; then
  echo usage: acceptance.sh source.xlsx [outdir]
  exit 2
fi
DIR=$(cd $(dirname $0) && pwd)
SRC=$1
OUT=/tmp/kaihu-dry
if [ $# -ge 2 ]; then OUT=$2; fi
rm -rf $OUT $OUT.2
python3 -m py_compile $DIR/*.py
A=$?
python3 $DIR/import_kaihu.py --dry-run --selfcheck --source $SRC --outdir $OUT
B=$?
python3 $DIR/import_kaihu.py --dry-run --selfcheck --source $SRC --outdir $OUT.2
C1=$?
M1=$(md5 -q $OUT/plan.json)
M2=$(md5 -q $OUT.2/plan.json)
echo acceptance_rc A_py_compile=$A B_selfcheck=$B C_dryrun2=$C1
echo plan_md5_1=$M1
echo plan_md5_2=$M2
if [ $M1 != $M2 ]; then
  echo C_deterministic_FAIL
  exit 1
fi
echo C_deterministic_PASS
if [ $A -ne 0 ] || [ $B -ne 0 ] || [ $C1 -ne 0 ]; then
  exit 1
fi
echo acceptance_ALL_PASS