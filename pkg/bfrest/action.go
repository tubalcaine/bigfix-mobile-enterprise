// This file is automatically generated. DO NOT EDIT.

package bfrest

type BES struct {
	SingleAction struct {
		ActionScript struct {
			MIMEType string `xml:"MIMEType,attr"`
			CharData string `xml:",chardata"`
		} `xml:"ActionScript"`
		IsUrgent  bool `xml:"IsUrgent"`
		Parameter struct {
			Name     string `xml:"Name,attr"`
			CharData string `xml:",chardata"`
		} `xml:"Parameter"`
		Relevance string `xml:"Relevance"`
		Settings  struct {
			ActiveUserRequirement   string `xml:"ActiveUserRequirement"`
			ActiveUserType          string `xml:"ActiveUserType"`
			ContinueOnErrors        bool   `xml:"ContinueOnErrors"`
			EndDateTimeLocalOffset  string `xml:"EndDateTimeLocalOffset"`
			HasDayOfWeekConstraint  bool   `xml:"HasDayOfWeekConstraint"`
			HasEndTime              bool   `xml:"HasEndTime"`
			HasReapplyInterval      bool   `xml:"HasReapplyInterval"`
			HasReapplyLimit         bool   `xml:"HasReapplyLimit"`
			HasRetry                bool   `xml:"HasRetry"`
			HasRunningMessage       bool   `xml:"HasRunningMessage"`
			HasStartTime            bool   `xml:"HasStartTime"`
			HasTemporalDistribution bool   `xml:"HasTemporalDistribution"`
			HasTimeRange            bool   `xml:"HasTimeRange"`
			HasWhose                bool   `xml:"HasWhose"`
			IsOffer                 bool   `xml:"IsOffer"`
			PostActionBehavior      struct {
				Behavior string `xml:"Behavior,attr"`
			} `xml:"PostActionBehavior"`
			PreActionCacheDownload bool `xml:"PreActionCacheDownload"`
			PreActionShowUI        bool `xml:"PreActionShowUI"`
			Reapply                bool `xml:"Reapply"`
			ReapplyLimit           int  `xml:"ReapplyLimit"`
			UseUTCTime             bool `xml:"UseUTCTime"`
		} `xml:"Settings"`
		SettingsLocks struct {
			ActionUITitle         bool `xml:"ActionUITitle"`
			ActiveUserRequirement bool `xml:"ActiveUserRequirement"`
			ActiveUserType        bool `xml:"ActiveUserType"`
			AnnounceOffer         bool `xml:"AnnounceOffer"`
			ContinueOnErrors      bool `xml:"ContinueOnErrors"`
			DayOfWeekConstraint   bool `xml:"DayOfWeekConstraint"`
			EndDateTimeOffset     bool `xml:"EndDateTimeOffset"`
			HasRunningMessage     bool `xml:"HasRunningMessage"`
			IsOffer               bool `xml:"IsOffer"`
			OfferCategory         bool `xml:"OfferCategory"`
			OfferDescriptionHTML  bool `xml:"OfferDescriptionHTML"`
			PostActionBehavior    struct {
				AllowCancel bool `xml:"AllowCancel"`
				Behavior    bool `xml:"Behavior"`
				Deadline    bool `xml:"Deadline"`
				Text        bool `xml:"Text"`
				Title       bool `xml:"Title"`
			} `xml:"PostActionBehavior"`
			PreAction struct {
				AskToSaveWork    bool `xml:"AskToSaveWork"`
				DeadlineBehavior bool `xml:"DeadlineBehavior"`
				ShowActionButton bool `xml:"ShowActionButton"`
				ShowCancelButton bool `xml:"ShowCancelButton"`
				ShowConfirmation bool `xml:"ShowConfirmation"`
				Text             bool `xml:"Text"`
			} `xml:"PreAction"`
			PreActionCacheDownload bool `xml:"PreActionCacheDownload"`
			PreActionShowUI        bool `xml:"PreActionShowUI"`
			Reapply                bool `xml:"Reapply"`
			ReapplyLimit           bool `xml:"ReapplyLimit"`
			RetryCount             bool `xml:"RetryCount"`
			RetryWait              bool `xml:"RetryWait"`
			RunningMessage         struct {
				Text bool `xml:"Text"`
			} `xml:"RunningMessage"`
			StartDateTimeOffset  bool `xml:"StartDateTimeOffset"`
			TemporalDistribution bool `xml:"TemporalDistribution"`
			TimeRange            bool `xml:"TimeRange"`
			Whose                bool `xml:"Whose"`
		} `xml:"SettingsLocks"`
		SuccessCriteria struct {
			Option string `xml:"Option,attr"`
		} `xml:"SuccessCriteria"`
		Target struct {
			ComputerID int `xml:"ComputerID"`
		} `xml:"Target"`
		Title string `xml:"Title"`
	} `xml:"SingleAction"`
}