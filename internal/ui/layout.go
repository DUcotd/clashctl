package ui

import (
	"fmt"
	"strings"
)

type feedbackTone int

const (
	feedbackNone feedbackTone = iota
	feedbackInfo
	feedbackSuccess
)

type pageFeedbackState struct {
	errorText   string
	messageText string
	messageTone feedbackTone
}

func (f *pageFeedbackState) clear() {
	if f == nil {
		return
	}
	f.errorText = ""
	f.messageText = ""
	f.messageTone = feedbackNone
}

func (f *pageFeedbackState) setError(msg string) {
	if f == nil {
		return
	}
	f.errorText = strings.TrimSpace(msg)
	f.messageText = ""
	f.messageTone = feedbackNone
}

func (f *pageFeedbackState) setInfo(msg string) {
	if f == nil {
		return
	}
	f.errorText = ""
	f.messageText = strings.TrimSpace(msg)
	f.messageTone = feedbackInfo
}

func (f *pageFeedbackState) setSuccess(msg string) {
	if f == nil {
		return
	}
	f.errorText = ""
	f.messageText = strings.TrimSpace(msg)
	f.messageTone = feedbackSuccess
}

func feedbackBlock(feedback pageFeedbackState) string {
	lines := make([]string, 0, 2)
	if feedback.errorText != "" {
		lines = append(lines, ErrorStyle.Render("错误: "+feedback.errorText))
	}
	if feedback.messageText != "" {
		switch feedback.messageTone {
		case feedbackSuccess:
			lines = append(lines, SuccessStyle.Render(feedback.messageText))
		case feedbackInfo:
			lines = append(lines, InfoStyle.Render(feedback.messageText))
		}
	}
	return strings.Join(lines, "\n")
}

func renderCard(header string, feedback pageFeedbackState, body, footer string) string {
	content := HeaderStyle.Render(header)
	if block := feedbackBlock(feedback); block != "" {
		content += "\n\n" + block
	}
	if body != "" {
		content += "\n\n" + body
	}
	if footer != "" {
		content += "\n\n" + HelpStyle.Render(footer)
	}
	return BoxStyle.Render(content)
}

const minViewportContentHeight = 5

func renderCardWithViewport(state viewportState, screen Screen, baseViewportSize func() (int, int), header string, feedback pageFeedbackState, body, footer string, selectedIndex int) string {
	if !state.vpReady {
		return renderCard(header, feedback, body, footer)
	}
	innerWidth, innerHeight := baseViewportSize()
	vp := state.vp
	vp.Width = innerWidth
	headerBlock := HeaderStyle.Render(header)
	feedbackBlock := feedbackBlock(feedback)
	footerBlock := HelpStyle.Render(footer)
	chromeHeight := lineCount(headerBlock) + lineCount(feedbackBlock) + lineCount(footerBlock) + 2
	contentHeight := max(minViewportContentHeight, innerHeight-chromeHeight)
	vp.Height = contentHeight
	vp.SetContent(body)
	if selectedIndex >= 0 {
		if selectedIndex < vp.YOffset {
			vp.SetYOffset(selectedIndex)
		} else if selectedIndex >= vp.YOffset+vp.Height {
			vp.SetYOffset(selectedIndex - vp.Height + 1)
		} else if off, ok := state.screenOffsets[screen]; ok {
			vp.SetYOffset(off)
		}
	} else if off, ok := state.screenOffsets[screen]; ok {
		vp.SetYOffset(off)
	}
	scrollHint := ""
	if vp.TotalLineCount() > vp.Height {
		scrollHint = InfoStyle.Render(fmt.Sprintf("位置 %d/%d", min(vp.YOffset+vp.Height, vp.TotalLineCount()), vp.TotalLineCount())) + "\n"
	}
	content := headerBlock
	if feedbackBlock != "" {
		content += "\n\n" + feedbackBlock
	}
	content += "\n\n" + vp.View()
	if footer != "" {
		content += "\n" + scrollHint + footerBlock
	}
	return BoxStyle.Render(content)
}

func renderStaticCard(state viewportState, screen Screen, baseViewportSize func() (int, int), header string, feedback pageFeedbackState, body, footer string) string {
	return renderCardWithViewport(state, screen, baseViewportSize, header, feedback, body, footer, -1)
}

func renderScrollablePage(state viewportState, screen Screen, baseViewportSize func() (int, int), header string, feedback pageFeedbackState, body, footer string) string {
	return renderCardWithViewport(state, screen, baseViewportSize, header, feedback, body, footer, -1)
}

func renderSelectablePage(state viewportState, screen Screen, baseViewportSize func() (int, int), header string, feedback pageFeedbackState, body, footer string, selectedIndex int) string {
	return renderCardWithViewport(state, screen, baseViewportSize, header, feedback, body, footer, selectedIndex)
}

func isQuitKey(msg teaKey) bool {
	key := msg.String()
	return key == "ctrl+c" || key == "q"
}

type teaKey interface {
	String() string
}

func shouldDismissHelp(key string) bool {
	return key == "esc" || key == "?" || key == "enter"
}

func handleQuitConfirm(key string, quitConfirm *bool) (quit bool, cancel bool) {
	if *quitConfirm {
		if key == "y" || key == "enter" {
			return true, false
		}
		if key == "n" || key == "esc" {
			*quitConfirm = false
			return false, true
		}
		return false, false
	}
	return false, false
}

func cardChromeHeight(header string, feedback pageFeedbackState, footer string, extraLines int) int {
	h := lineCount(HeaderStyle.Render(header))
	h += lineCount(feedbackBlock(feedback))
	h += lineCount(HelpStyle.Render(footer))
	h += 2
	h += extraLines
	return h
}
