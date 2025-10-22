package util

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
)

const (
	firstIndex = 0
)

// GetGoroutineID 获取当前协程的ID
// 返回值: int - 当前协程的唯一标识ID
func GetGoroutineID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	var goroutineID int
	fmt.Sscanf(string(buf[:n]), "goroutine %d ", &goroutineID)
	return goroutineID
}

// GetIndexSlice 在整型切片中查找目标值的索引位置
// 参数:
//
//	target - 要查找的目标整数
//	slice - 整型切片
//
// 返回值: int - 目标值在切片中的索引位置，未找到返回-1
func GetIndexSlice(target int, slice []int) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}

// InSlice 判断元素是否在切片中（泛型版本）
// 参数:
//
//	slice - 切片
//	item - 要查找的元素
//
// 返回值: bool - 存在返回true，否则返回false
func InSlice[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Inarray 判断某个值是否存在于数组/切片中
// 参数:
//
//	val - 要查找的值（任意类型）
//	array - 数组/切片（任意类型）
//
// 返回值:
//
//	exists - bool，值是否存在
//	index - int，值在数组中的索引位置，不存在则为-1
func Inarray(val interface{}, array interface{}) (exists bool, index int) {
	if reflect.TypeOf(array).Kind() != reflect.Slice {
		return false, -1
	}

	s := reflect.ValueOf(array)
	for i := 0; i < s.Len(); i++ {
		if reflect.DeepEqual(val, s.Index(i).Interface()) {
			return true, i
		}
	}
	return false, -1
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail 验证邮箱格式是否合法
// 参数:
//
//	email - 待验证的邮箱字符串
//
// 返回值: bool - true表示邮箱格式合法，false表示不合法
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// IsValidUsername 验证用户名格式是否合法
// 规则：长度3-15个字符，只能包含字母、数字和下划线
// 参数:
//
//	username - 待验证的用户名字符串
//
// 返回值: bool - true表示用户名格式合法，false表示不合法
func IsValidUsername(username string) bool {
	length := len(username)
	if length < 3 || length > 15 {
		return false
	}

	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return false
		}
	}
	return true
}

// GetLastDateOfNextMonth 获取下个月的最后一天0点时间
// 参数:
//
//	d - 基准时间
//
// 返回值: time.Time - 下个月最后一天的0点时间
func GetLastDateOfNextMonth(d time.Time) time.Time {
	return GetFirstDateOfMonth(d).AddDate(0, 2, -1)
}

// ArrayUnique 对数组/切片进行去重（需要先排序）
// 参数:
//
//	a - 任意类型的数组/切片
//
// 返回值: []interface{} - 去重后的切片
func ArrayUnique(a interface{}) []interface{} {
	va := reflect.ValueOf(a)
	if va.Kind() != reflect.Slice {
		return nil
	}

	ret := make([]interface{}, 0, va.Len())
	for i := 0; i < va.Len(); i++ {
		if i > 0 && reflect.DeepEqual(va.Index(i-1).Interface(), va.Index(i).Interface()) {
			continue
		}
		ret = append(ret, va.Index(i).Interface())
	}
	return ret
}

// GenerateRandomNumbers 生成指定数量的不重复随机数
// 参数:
//
//	start - 随机数范围起始值（包含）
//	end - 随机数范围结束值（不包含）
//	count - 需要生成的随机数数量
//
// 返回值: []int - 生成的随机数切片，范围检查失败返回nil
func GenerateRandomNumbers(start, end, count int) []int {
	// 范围检查
	if end <= start || (end-start) < count || count < 0 {
		return nil
	}

	// 存放结果的slice
	nums := make([]int, 0, count)
	numSet := make(map[int]bool, count)

	// 随机数生成器，加入时间戳保证每次生成的随机数不一样
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for len(nums) < count {
		num := r.Intn(end-start) + start
		if !numSet[num] {
			numSet[num] = true
			nums = append(nums, num)
		}
	}

	return nums
}

// GenerateRandomNumber 生成一个随机数
// 参数:
//
//	start - 随机数范围起始值（包含）
//	end - 随机数范围结束值（不包含）
//
// 返回值: int - 生成的随机数
func GenerateRandomNumber(start, end int) int {
	if end <= start {
		return start
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(end-start) + start
}

// 已废弃：使用 GenerateRandomNumber 代替
// Deprecated: Use GenerateRandomNumber instead
func GenerateRandomSingleNumber(start int, end int, count int) int {
	if end < start || (end-start) < count {
		return 0
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Intn(end-start) + start
}

// 已废弃：使用 GenerateRandomNumbers 代替
// Deprecated: Use GenerateRandomNumbers instead
func GenerateRandom(start int, end int, count int) []int {
	return GenerateRandomNumbers(start, end, count)
}

// Wait 等待指定的秒数
// 参数:
//
//	seconds - 等待的秒数（支持小数，如0.5表示500毫秒）
func Wait(seconds float32) {
	time.Sleep(time.Duration(seconds * float32(time.Second)))
}

// StrToMap 将逗号分隔的字符串转换为map[int]int
// 参数:
//
//	s - 逗号分隔的数字字符串，如"1,2,3"
//
// 返回值: map[int]int - 转换后的map（key和value相同）
func StrToMap(s string) map[int]int {
	if s == "" {
		return make(map[int]int)
	}

	tmpList := strings.Split(s, ",")
	out := make(map[int]int, len(tmpList))

	for _, v := range tmpList {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if tmpid, err := strconv.Atoi(v); err == nil {
			out[tmpid] = tmpid
		}
	}
	return out
}

// StrToInt 将逗号分隔的字符串转换为整型切片
// 参数:
//
//	s - 逗号分隔的数字字符串，如"1,2,3"
//
// 返回值: []int - 转换后的整型切片
func StrToInt(s string) []int {
	if s == "" {
		return []int{}
	}

	tmpList := strings.Split(s, ",")
	out := make([]int, 0, len(tmpList))

	for _, v := range tmpList {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if tmpid, err := strconv.Atoi(v); err == nil {
			out = append(out, tmpid)
		}
	}
	return out
}

// IntSliceToStringSlice 将整型切片转换为字符串切片
// 参数:
//
//	target - 整型切片
//
// 返回值: []string - 转换后的字符串切片
func IntSliceToStringSlice(target []int) []string {
	out := make([]string, 0, len(target))
	for _, v := range target {
		out = append(out, strconv.Itoa(v))
	}
	return out
}

// StringToInt64 将逗号分隔的字符串转换为int64切片
// 参数:
//
//	str - 逗号分隔的数字字符串，如"1,2,3"
//
// 返回值: []int64 - 转换后的int64切片
func StringToInt64(str string) []int64 {
	if str == "" {
		return []int64{}
	}

	tmpList := strings.Split(str, ",")
	out := make([]int64, 0, len(tmpList))

	for _, v := range tmpList {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if tmpId, err := strconv.ParseInt(v, 10, 64); err == nil {
			out = append(out, tmpId)
		}
	}
	return out
}

// InArray 判断某个值是否存在于数组/切片中（带错误返回）
// 参数:
//
//	search - 要查找的值（任意类型）
//	haystack - 数组/切片（任意类型）
//
// 返回值:
//
//	exists - bool，值是否存在
//	index - int，值在数组中的索引位置，不存在则为-1
//	err - error，类型不支持时返回错误信息
func InArray(search interface{}, haystack interface{}) (bool, int, error) {
	if reflect.TypeOf(haystack).Kind() != reflect.Slice {
		return false, -1, errors.New("unsupported type: haystack must be a slice")
	}

	s := reflect.ValueOf(haystack)
	for i := 0; i < s.Len(); i++ {
		if reflect.DeepEqual(search, s.Index(i).Interface()) {
			return true, i, nil
		}
	}
	return false, -1, nil
}

// IntSliceToStrSlice 将任意整型切片转换为字符串切片
// 支持int8, int16, int32, int64, int类型
// 参数:
//
//	data - 整型切片（支持多种整型）
//
// 返回值:
//
//	out - []string，转换后的字符串切片
//	err - error，类型不支持时返回错误信息
func IntSliceToStrSlice(data interface{}) ([]string, error) {
	if reflect.TypeOf(data).Kind() != reflect.Slice {
		return nil, errors.New("unsupported type: data must be a slice")
	}

	s := reflect.ValueOf(data)
	out := make([]string, 0, s.Len())

	for i := 0; i < s.Len(); i++ {
		var tmpData int64
		val := s.Index(i).Interface()

		switch v := val.(type) {
		case int8:
			tmpData = int64(v)
		case int16:
			tmpData = int64(v)
		case int32:
			tmpData = int64(v)
		case int64:
			tmpData = v
		case int:
			tmpData = int64(v)
		default:
			return nil, errors.New("unsupported element type: must be int type")
		}

		out = append(out, strconv.FormatInt(tmpData, 10))
	}
	return out, nil
}

// RemoveDuplicate 移除整型切片中的重复元素（保持原顺序）
// 参数:
//
//	list - 整型切片
//
// 返回值: []int - 去重后的整型切片
func RemoveDuplicate(list []int) []int {
	if len(list) == 0 {
		return []int{}
	}

	seen := make(map[int]bool, len(list))
	out := make([]int, 0, len(list))

	for _, v := range list {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// GetFirstDateOfMonth 获取传入时间所在月份的第一天0点时间
// 参数:
//
//	d - 基准时间
//
// 返回值: time.Time - 该月第一天的0点时间
func GetFirstDateOfMonth(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, d.Location())
}

// GetLastDateOfMonth 获取传入时间所在月份的最后一天0点时间
// 参数:
//
//	d - 基准时间
//
// 返回值: time.Time - 该月最后一天的0点时间
func GetLastDateOfMonth(d time.Time) time.Time {
	return GetFirstDateOfMonth(d).AddDate(0, 1, -1)
}

// GetZeroTime 获取某一天的0点时间（00:00:00）
// 参数:
//
//	d - 基准时间
//
// 返回值: time.Time - 该天的0点时间
func GetZeroTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

// GetEndTime 获取某一天的结束时间（23:59:59）
// 参数:
//
//	d - 基准时间
//
// 返回值: time.Time - 该天的23:59:59时间
func GetEndTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, 0, d.Location())
}

// IntersectionInt 获取两个整型切片的交集
// 参数:
//
//	slice1 - 第一个整型切片
//	slice2 - 第二个整型切片
//
// 返回值: []int - 两个切片的交集
func IntersectionInt(slice1, slice2 []int) []int {
	if len(slice1) == 0 || len(slice2) == 0 {
		return []int{}
	}

	// 使用map提高查找效率
	set := make(map[int]bool, len(slice2))
	for _, v := range slice2 {
		set[v] = true
	}

	result := make([]int, 0)
	seen := make(map[int]bool) // 避免重复添加

	for _, v := range slice1 {
		if set[v] && !seen[v] {
			result = append(result, v)
			seen[v] = true
		}
	}

	return result
}

// TimeParse 将字符串按指定格式解析为本地时间
// 参数:
//
//	layout - 时间格式，如"2006-01-02 15:04:05"
//	in - 待解析的时间字符串
//
// 返回值:
//
//	time.Time - 解析后的时间对象
//	error - 解析错误
func TimeParse(layout, in string) (time.Time, error) {
	local, err := time.LoadLocation("Local")
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation(layout, in, local)
}
