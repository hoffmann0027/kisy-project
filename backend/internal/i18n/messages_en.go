package i18n

// #nosec G101 -- interface text keyed by topic ("auth.passwordRule"), not a credential
var en = Catalog{
	// --- auth ---
	"auth.captchaUnavailable": "Verification is unavailable right now. Try again in a minute",
	"auth.captchaFailed":      "We couldn't confirm you're not a robot. Reload the page and try again",
	"auth.passwordRule":       "password: 12–128 characters, at least one letter and one digit",

	// --- people and chats ---
	"blocks.self":                 "You can't block yourself",
	"reports.self":                "You can't report yourself",
	"chats.restricted":            "This person has restricted who can message them",
	"messages.encryptionRequired": "Private messages can only be sent encrypted. Please update the app.",

	// --- account ---
	"users.deleteWord":        "DELETE",
	"users.deleteConfirm":     "type %s to confirm deletion",
	"users.wrongPassword":     "wrong password",
	"users.ownerCannotDelete": "The owner's account can't be deleted: hand over ownership first",
	"users.nameTaken":         "This name is taken",
	"users.nameChars":         "Letters only, with single spaces between words",
	"users.nameLength":        "Name: 2 to 40 characters",

	// --- groups and communities ---
	"groups.onlyCeoLevel":       "Only the CEO can change a group's level",
	"groups.sanctionedDelete":   "This community is under moderation sanctions and can't be deleted while they're in effect",
	"groups.levelAboveYours":    "You can't create a group with an access level above your own",
	"groups.membersByOwnerOnly": "Only the group owner adds members; people join communities themselves",
	"posts.fileTooLarge":        "The file is too large",
	"posts.membersOnly":         "This community is private: only members can see its posts",
	"posts.noRights":            "You don't have permission to post here",
	"posts.notCommunity":        "This is a group, not a community",
	"posts.empty":               "A post can't be empty",
	"posts.tooLong":             "The post is too long",
	"boards.defaultTitle":       "Task board",
	"boards.columnTodo":         "To do",
	"boards.columnDoing":        "In progress",
	"boards.columnDone":         "Done",

	// --- moderation ---
	"moderation.reasonRequired":     "Give a reason",
	"moderation.reasonTooLong":      "The reason can be at most 1000 characters",
	"moderation.restoreFirst":       "The community has been deleted — restore it first",
	"moderation.notDeleted":         "The community isn't deleted",
	"moderation.sanctionPermanent":  "This sanction can't be lifted",
	"moderation.restoreExpired":     "The restore period has expired",
	"moderation.warn.community":     "The community “%s” has received a warning (%d of %d): %s",
	"moderation.warn.group":         "The group “%s” has received a warning (%d of %d): %s",
	"moderation.mute.community":     "The community “%s” is muted %s — its posts are hidden from the feed: %s",
	"moderation.mute.group":         "The group “%s” is muted %s — its posts are hidden from the feed: %s",
	"moderation.delete.community":   "The community “%s” has been deleted: %s",
	"moderation.delete.group":       "The group “%s” has been deleted: %s",
	"moderation.restored.community": "The community “%s” has been restored",
	"moderation.restored.group":     "The group “%s” has been restored",
	"moderation.untilForever":       "indefinitely",
	"moderation.until":              "until %s (UTC)",

	// --- limits on new accounts and storage ---
	"quarantine.posts":         "Posting opens %s",
	"quarantine.communities":   "Creating communities opens %s",
	"quarantine.newChats":      "For now you can start only a few new chats a day. The limit lifts %s",
	"quarantine.upload":        "New accounts can send only smaller files. The limit lifts %s",
	"quarantine.other":         "This feature opens %s",
	"quarantine.inAnHour":      "in an hour",
	"quarantine.inHours#one":   "in %d hour",
	"quarantine.inHours#other": "in %d hours",
	"quota.userStorage":        "You're out of space for files: delete old attachments or notes",
	"quota.communityStorage":   "This community is out of space for media",
	"quota.postRate":           "Too many posts this hour — try again later",

	// --- feedback and announcements ---
	"feedback.nextSoon":        "You can leave feedback once a day. The next one in less than an hour",
	"feedback.nextInHours":     "You can leave feedback once a day. The next one in %d h",
	"feedback.replyPushTitle":  "Reply to your feedback",
	"announcements.dailyLimit": "Daily limit reached: at most %d broadcasts and %d personal notifications per 24 hours",
	"releases.pushTitle":       "Version %s is out",

	// --- pushes ---
	"push.mentioned":  "You were mentioned in a message",
	"push.newMessage": "New message",

	// --- admin ---
	"dashboard.down":          "not responding",
	"dashboard.objectStorage": "object storage",
	"dashboard.inDatabase":    "in the database",
	"rating.csv.date":         "Date",
	"rating.csv.project":      "Project",
	"rating.csv.task":         "Task",
	"rating.csv.income":       "Income",
	"rating.csv.expense":      "Expense",
	"rating.csv.profit":       "Profit",
	"rating.csv.author":       "Author",
	"rating.csv.comment":      "Comment",
}
