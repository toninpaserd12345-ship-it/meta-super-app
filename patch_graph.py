import re

with open('backend/internal/infrastructure/meta/graph_connector.go', 'r') as f:
    content = f.read()

# Add Recipient to struct
old_struct = """				Sender struct {
					ID string `json:"id"`
				} `json:"sender"`
				Timestamp int64 `json:"timestamp"`"""
new_struct = """				Sender struct {
					ID string `json:"id"`
				} `json:"sender"`
				Recipient struct {
					ID string `json:"id"`
				} `json:"recipient"`
				Timestamp int64 `json:"timestamp"`"""
content = content.replace(old_struct, new_struct)

# Update the webhook logic
old_logic = """			if message.Sender.ID == "" {
				continue
			}
			accountID, page, pageFound := g.findPage(entry.ID)
			if !pageFound {
				continue
			}
			if g.leadSink != nil {
				sourceType, sourceID := "message", ""
				if ref != nil {
					sourceType, sourceID = "post", ref.Ref
					if ref.AdID != "" {
						sourceType, sourceID = "ad", ref.AdID
					}
				}
				_ = g.leadSink.CaptureLead(ctx, domain.LeadCapture{AccountID: accountID, PageID: entry.ID, SenderID: message.Sender.ID, SourceType: sourceType, SourceID: sourceID})
			}"""

new_logic = """			if message.Sender.ID == "" {
				continue
			}
			accountID, page, pageFound := g.findPage(entry.ID)
			if !pageFound {
				continue
			}
			
			customerID := message.Sender.ID
			isSystem := false
			if message.Sender.ID == entry.ID {
				customerID = message.Recipient.ID
				isSystem = true
			}
			
			if g.leadSink != nil && !isSystem {
				sourceType, sourceID := "message", ""
				if ref != nil {
					sourceType, sourceID = "post", ref.Ref
					if ref.AdID != "" {
						sourceType, sourceID = "ad", ref.AdID
					}
				}
				_ = g.leadSink.CaptureLead(ctx, domain.LeadCapture{AccountID: accountID, PageID: entry.ID, SenderID: customerID, SourceType: sourceType, SourceID: sourceID})
			}"""
content = content.replace(old_logic, new_logic)

# Update the broadcast logic
old_broadcast = """						senderName, senderPic := g.GetProfile(ctx, message.Sender.ID, page.AccessToken)
						g.chatStream.Broadcast(accountID, usecase.ChatEvent{
							AccountID:  accountID,
							PageID:     entry.ID,
							SenderID:   message.Sender.ID,
							SenderName: senderName,
							SenderPic:  senderPic,
							Message:    msgContent,
							Type:       msgType,
							Timestamp:  timestampStr,
							Platform:   "facebook",
						})"""

new_broadcast = """						senderName, senderPic := g.GetProfile(ctx, customerID, page.AccessToken)
						platform := "facebook"
						if isSystem {
							platform = "system"
						}
						g.chatStream.Broadcast(accountID, usecase.ChatEvent{
							AccountID:  accountID,
							PageID:     entry.ID,
							SenderID:   customerID,
							SenderName: senderName,
							SenderPic:  senderPic,
							Message:    msgContent,
							Type:       msgType,
							Timestamp:  timestampStr,
							Platform:   platform,
						})"""
content = content.replace(old_broadcast, new_broadcast)

# Fix seenKey using customerID
content = content.replace("seenKey := entry.ID + \":\" + message.Sender.ID", "seenKey := entry.ID + \":\" + customerID")

with open('backend/internal/infrastructure/meta/graph_connector.go', 'w') as f:
    f.write(content)
