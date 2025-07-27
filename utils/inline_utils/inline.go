package inline_utils

import (
	"fmt"
	"strconv"
	"strings"
	"trbot/utils/configs"
	"trbot/utils/type/contain"

	"github.com/go-telegram/bot/models"
)

// 将 InlineQueryResult 列表进行分页处理
func ResultPagination(fields []string, results []models.InlineQueryResult) []models.InlineQueryResult {
	// 当 result 的数量超过 InlineResultsPerPage 时，进行分页
	// fmt.Println(len(results), InlineResultsPerPage)
	if len(results) > configs.BotConfig.InlineResultsPerPage {
		// 获取 update.InlineQuery.Query 末尾的 `<分页符号><数字>` 来选择输出第几页
		var pageNow int = 1
		var pageSize = (configs.BotConfig.InlineResultsPerPage - 1)

		pageNow, err := ExtractPageNumber(fields)
		// 读取页码发生错误
		if err != nil {
			// 输入了分页符号没有输入数字
			if fields[len(fields)-1][1:] == "" {
				return []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "keepInputNumber",
					Title:       "请继续输入数字",
					Description: fmt.Sprintf("继续输入一个数字来查看对应的页面，当前列表有 %d 页", (len(results) + pageSize - 1) / pageSize),
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "用户在尝试进行分页时点击了分页提示...",
						ParseMode:   models.ParseModeMarkdownV1,
					},
				}}
			} else {
				// 在分页符号后输入了非数字字符
				return []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "noThisOperation",
					Title:       "无效的操作",
					Description: fmt.Sprintf("若您想翻页查看，请尝试输入 `%s2` 来查看第二页", configs.BotConfig.InlinePaginationSymbol),
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "用户在尝试进行分页时输入了错误的页码并点击了分页提示...",
						ParseMode:   models.ParseModeMarkdownV1,
					},
				}}
			}
		}

		start := (pageNow - 1) * pageSize
		end := start + pageSize

		if start < 0 || start >= len(results) {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "wrongPageNumber",
				Title:       "错误的页码",
				Description: fmt.Sprintf("您输入的页码 %d 超出范围，当前列表有 %d 页", pageNow, (len(results)+pageSize-1)/pageSize),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在浏览不存在的页面时点击了错误页码提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			}}
		}

		if end > len(results) {
			end = len(results)
		}
		pageResults := results[start:end]

		// 添加翻页提示
		if end < len(results) {
			totalPages := (len(results) + pageSize - 1) / pageSize
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", totalPages-pageNow, configs.BotConfig.InlinePaginationSymbol, pageNow+1),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在挑选内容时点击了分页提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			})
		} else {
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: "后面已经没有东西了",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在挑选内容时点击了分页提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			})
		}

		return pageResults
	} else if len(fields) > 0 && strings.HasPrefix(fields[len(fields)-1], configs.BotConfig.InlinePaginationSymbol) {
		return []models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID:          "noNeedPagination",
			Title:       "没有多余的内容",
			Description: fmt.Sprintf("只有 %d 个条目，你想翻页也没有多的了", len(results)),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
				ParseMode:   models.ParseModeMarkdownV1,
			},
		}}
	} else {
		return results
	}
}

// 从 inline 字段中提取子命令字符串
func ExtractSubCommand(fields []string) string {
	if len(fields) == 0 {
		return ""
	}

	// 判断是不是子命令
	if strings.HasPrefix(fields[0], configs.BotConfig.InlineSubCommandSymbol) {
		return strings.TrimPrefix(fields[0], configs.BotConfig.InlineSubCommandSymbol)
	}
	return ""
}

// 从 Inline 字段中提取查询关键词，去除子命令的前缀或后缀的分页符号
func ExtractKeywords(fields []string) []string {
	if len(fields) == 0 {
		return []string{}
	}

	// 判断是不是子命令
	if strings.HasPrefix(fields[0], configs.BotConfig.InlineSubCommandSymbol) {
		fields = fields[1:]
	}
	// 判断有没有分页符号
	if len(fields) > 0 && strings.HasPrefix(fields[len(fields)-1], configs.BotConfig.InlinePaginationSymbol) {
		fields = fields[:len(fields)-1]
	}

	return fields
}

// 从 inline 字段中提取页码
func ExtractPageNumber(fields []string) (int, error) {
	if len(fields) == 0 {
		return 1, nil
	}

	// 判断有没有分页符号
	if strings.HasPrefix(fields[len(fields)-1], configs.BotConfig.InlinePaginationSymbol) {
		return strconv.Atoi(fields[len(fields)-1][1:])
	}
	return 1, nil
}

// 从 inline 查询字段中匹配多个关键词
func MatchMultKeyword(fields []string, keywords []string) bool {
	var allkeywords int

	fields = ExtractKeywords(fields)
	if len(fields) != 0 {
		allkeywords = len(fields)
	}
	// fmt.Println(allkeywords)
	if allkeywords == 1 {
		if len(keywords) == 0 {
			return false
		}
		if contain.SubStringCaseInsensitive(fields[0], keywords...) {
			return true
		}
	} else {
		var allMatch bool = true

		for _, n := range fields {
			if contain.SubStringCaseInsensitive(n, keywords...) {
				// 保持 current 内容，继续过滤
				// continue
			} else {
				// 只要有一个关键词未匹配，返回 false
				allMatch = false
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}
