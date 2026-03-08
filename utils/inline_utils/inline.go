package inline_utils

import (
	"fmt"
	"strconv"
	"strings"

	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/type/contain"

	"github.com/go-telegram/bot/models"
)

type ParsedQuery struct {
	SubCommand string
	Keywords   []string
	Category   string
	Page       int
	LastChar   string
}

// group Keywords as a string
func (pq ParsedQuery)KeywordQuery() (query string) {
	for _, n := range pq.Keywords {
		query += n + " "
	}
	return
}

func ParseInlineFields(fields []string) ParsedQuery {
	result := ParsedQuery{}

	for i, field := range fields {
		switch {
		case i == 0 && strings.HasPrefix(field, configs.BotConfig.InlineSubCommandSymbol):
			result.SubCommand = strings.TrimPrefix(field, configs.BotConfig.InlineSubCommandSymbol)
		case strings.HasPrefix(field, configs.BotConfig.InlineCategorySymbol):
			catStr := strings.TrimPrefix(field, configs.BotConfig.InlineCategorySymbol)
			if catStr != "" {
				if result.Category != "" {
					// previous category was not empty, so it's a keyword
					result.Keywords = append(result.Keywords, configs.BotConfig.InlineCategorySymbol + result.Category)
				}
				result.Category = catStr
			} else if i + 1 == len(fields) {
				result.LastChar = field
			} else {
				result.Keywords = append(result.Keywords, field)
			}
		case strings.HasPrefix(field, configs.BotConfig.InlinePaginationSymbol):
			pageStr := strings.TrimPrefix(field, configs.BotConfig.InlinePaginationSymbol)
			if pageNum, err := strconv.Atoi(pageStr); err == nil {
				if result.Page != 0 {
					// previous page was not empty, so it's a keyword
					result.Keywords = append(result.Keywords, configs.BotConfig.InlinePaginationSymbol + strconv.Itoa(result.Page))
				}
				result.Page = pageNum
			} else if i + 1 == len(fields) {
				result.LastChar = field
			} else {
				result.Keywords = append(result.Keywords, field)
			}
		default:
			result.Keywords = append(result.Keywords, field)
			if i + 1 == len(fields) && len(field) == 1 {
				result.LastChar = field
			}
		}
	}

	if result.Page == 0 {
		result.Page = 1
	}

	// zerolog/log
	// log.Warn().
	// 	Interface("parsedQuery", result).
	// 	Msg("Parsed query")

	return result
}

func ParseInlineQuery(query string) ParsedQuery {
	return ParseInlineFields(strings.Fields(query))
}

// 将 InlineQueryResult 列表进行分页处理
func ResultPagination(parsedQuery ParsedQuery, results []models.InlineQueryResult) []models.InlineQueryResult {
	var pageSize    int = (configs.BotConfig.InlineResultsPerPage - 1)
	var resultCount int = len(results)

	if parsedQuery.LastChar != "" {
		switch parsedQuery.LastChar {
		case configs.BotConfig.InlinePaginationSymbol: // 最后一个字符为分页符号
			if resultCount < configs.BotConfig.InlineResultsPerPage {
				return append([]models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "noNeedPagination",
					Title:       "没有多余的内容",
					Description: fmt.Sprintf("只有以下 %d 个条目", resultCount),
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
				}}, results...)
			} else {
				return []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "keepInputNumber",
					Title:       "请继续输入数字",
					Description: fmt.Sprintf("继续输入一个数字来查看对应的页面，当前列表有 %d 页", (resultCount + pageSize - 1) / pageSize),
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试进行分页时点击了提示信息..." },
				}}
			}
		case configs.BotConfig.InlineCategorySymbol: // 最后一个字符为分类符号
			results = append([]models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "noCategory",
				Title:       "没有分类",
				Description: "这个插件并没有设定任何分类",
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试选择分类时点击了提示信息..." },
			}}, results...)
		}
	}

	// 当 result 的数量超过 InlineResultsPerPage 时，进行分页
	if resultCount > configs.BotConfig.InlineResultsPerPage {
		var pageSize = (configs.BotConfig.InlineResultsPerPage - 1)

		start := (parsedQuery.Page - 1) * pageSize
		end   := start + pageSize

		if start < 0 || start >= resultCount {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "wrongPageNumber",
				Title:       "错误的页码",
				Description: fmt.Sprintf("您输入的页码 %d 超出范围，当前列表有 %d 页", parsedQuery.Page, (resultCount + pageSize - 1) / pageSize),
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在浏览不存在的页面时点击了错误页码提示..." },
			}}
		}

		if end > resultCount { end = resultCount }
		pageResults := results[start:end]

		// 添加翻页提示
		if end < resultCount {
			totalPages := (resultCount + pageSize - 1) / pageSize
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", parsedQuery.Page),
				Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", totalPages - parsedQuery.Page, configs.BotConfig.InlinePaginationSymbol, parsedQuery.Page + 1),
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在挑选内容时点击了分页提示..." },
			})
		} else {
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", parsedQuery.Page),
				Description: "后面已经没有东西了",
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在挑选内容时点击了分页提示..." },
			})
		}

		return pageResults
	} else {
		if parsedQuery.Page > 1 {
			return append([]models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "noNeedPagination",
				Title:       "没有多余的内容",
				Description: fmt.Sprintf("只有以下 %d 个条目", resultCount),
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
			}}, results...)
		}
		return results
	}
}

// include ResultPagination
func ResultCategory(parsedQuery ParsedQuery, categoryResults map[string][]models.InlineQueryResult) []models.InlineQueryResult {
	var resultCount int

	var categorys []string
	for name, results := range categoryResults {
		if len(results) > 0 {
			categorys = append(categorys, name)
			resultCount += len(results)
		}
	}

	if parsedQuery.LastChar == configs.BotConfig.InlineCategorySymbol {
		// 最后一个字符为分类符号
		return []models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID:          "keepInputCategory",
			Title:       "请继续输入分类名称",
			Description: fmt.Sprintf("当前列表有 %v 分类", categorys),
			InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试选择分类时点击了提示信息..." },
		}}
	}

	if parsedQuery.Category != "" {
		result, IsExist := categoryResults[parsedQuery.Category]
		if !IsExist || len(result) == 0 {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "noThisCategory",
				Title:       fmt.Sprintf("无效的 [ %s ] 分类", parsedQuery.Category),
				Description: fmt.Sprintf("当前列表有 %s 分类", categorys),
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试访问不存在的分类时点击了提示信息..." },
			}}
		}

		return ResultPagination(parsedQuery, result)
	} else {
		var allResults []models.InlineQueryResult
		for _, result := range categoryResults {
			allResults = append(allResults, result...)
		}
		return ResultPagination(parsedQuery, allResults)
	}
}

// 从 inline 查询字段中匹配多个关键词
func MatchMultKeyword(targetKeywords []string, keywords []string) bool {
	var allkeywords int

	if len(targetKeywords) != 0 {
		allkeywords = len(targetKeywords)
	}
	// fmt.Println(allkeywords)
	if allkeywords == 1 {
		if len(keywords) == 0 {
			return false
		}
		if contain.SubStringCaseInsensitive(targetKeywords[0], keywords...) {
			return true
		}
	} else {
		var allMatch bool = true

		for _, n := range targetKeywords {
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
