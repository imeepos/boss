// 检查 D:迁移编号唯一性(防并行 worktree 各自"最大号+1"撞号,2026-08-22 两次撞号后固化)。
// D1 同树撞号:同一数字前缀对应多个不同迁移名(排除 .up/.down 成对同名)。
// D2 跨分支撞号:未合并本地分支占用了本树已在用的编号且迁移名不同。
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// checkMigrations 返回失败数。
func checkMigrations(root string) int {
	fails := 0
	mine := scanMigrationNames(root)
	fails += reportTreeDuplicates(mine)
	fails += reportBranchCollisions(root, mine)
	if fails == 0 {
		fmt.Println("D OK 迁移编号无同树/跨未合并分支撞号")
	}
	return fails
}

// scanMigrationNames 扫描 migrations/ 下的 编号→迁移名集合(剥离 .up/.down 后缀)。
func scanMigrationNames(root string) map[int]map[string]bool {
	out := map[int]map[string]bool{}
	entries, err := os.ReadDir(filepath.Join(root, "migrations"))
	if err != nil {
		return out
	}
	for _, e := range entries {
		name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".up.sql"), ".down.sql")
		dash := strings.Index(name, "_")
		if dash <= 0 {
			continue
		}
		n, err := strconv.Atoi(name[:dash])
		if err != nil {
			continue
		}
		if out[n] == nil {
			out[n] = map[string]bool{}
		}
		out[n][name] = true
	}
	return out
}

// reportTreeDuplicates D1:同编号多个迁移名即撞号(存量历史撞号走 baseline 豁免)。
func reportTreeDuplicates(mine map[int]map[string]bool) int {
	fails := 0
	base := loadBaseline()
	nums := sortedKeys(mine)
	for _, n := range nums {
		if len(mine[n]) > 1 && !base[fmt.Sprintf("mig:%06d", n)] {
			names := sortedSet(mine[n])
			fmt.Printf("D FAIL 同树撞号: %06d 被 %s 占用\n", n, strings.Join(names, " / "))
			fails++
		}
	}
	return fails
}

// reportBranchCollisions D2:遍历未合并本地分支,其迁移编号与本树同名冲突即撞号。
// 已合并分支(本树已含其迁移,即使历史名不同)跳过,避免改名历史误报。
func reportBranchCollisions(root string, mine map[int]map[string]bool) int {
	fails := 0
	for _, br := range unmergedBranches(root) {
		for name := range branchMigrationNames(root, br) {
			dash := strings.Index(name, "_")
			if dash <= 0 {
				continue
			}
			n, err := strconv.Atoi(name[:dash])
			if err != nil || mine[n] == nil {
				continue
			}
			if !mine[n][name] {
				fmt.Printf("D FAIL 跨分支撞号: %s 在未合并分支 %s 占用 %06d(本树为 %s)\n",
					name, br, n, strings.Join(sortedSet(mine[n]), " / "))
				fails++
			}
		}
	}
	return fails
}

// unmergedBranches 非 HEAD 祖先的本地分支(即有未合并工作)。
func unmergedBranches(root string) []string {
	out, err := gitLines(root, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil
	}
	var branches []string
	for _, br := range out {
		if br == "" {
			continue
		}
		if isAncestor := gitOK(root, "merge-base", "--is-ancestor", br, "HEAD"); isAncestor {
			continue
		}
		branches = append(branches, br)
	}
	return branches
}

// branchMigrationNames 指定分支 migrations/ 下全部文件名(已剥 .up/.down)。
func branchMigrationNames(root, branch string) map[string]bool {
	out := map[string]bool{}
	lines, err := gitLines(root, "ls-tree", "-r", "--name-only", branch, "--", "migrations/")
	if err != nil {
		return nil
	}
	for _, f := range lines {
		name := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(f), ".up.sql"), ".down.sql")
		if name != "" {
			out[name] = true
		}
	}
	return out
}

func gitLines(root string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n"), nil
}

func gitOK(root string, args ...string) bool {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	return cmd.Run() == nil
}

func sortedKeys(m map[int]map[string]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
