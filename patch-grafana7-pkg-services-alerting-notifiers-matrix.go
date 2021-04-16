--- /dev/null
+++ pkg/services/alerting/notifiers/matrix.go
@@ -0,0 +1,128 @@
+package notifiers
+
+import (
+	"fmt"
+	"github.com/grafana/grafana/pkg/bus"
+	"github.com/grafana/grafana/pkg/components/simplejson"
+	"github.com/grafana/grafana/pkg/infra/log"
+	"github.com/grafana/grafana/pkg/models"
+	"github.com/grafana/grafana/pkg/services/alerting"
+)
+
+func init() {
+	alerting.RegisterNotifier(&alerting.NotifierPlugin{
+		Type:        "matrix",
+		Name:        "Matrix",
+		Description: "Sends notifications to Matrix room",
+		Heading:     "Matrix settings",
+		Factory:     NewMatrixNotifier,
+                Options: []alerting.NotifierOption{
+                        {
+                                Label:        "Matrix room URL",
+                                Element:      alerting.ElementTypeInput,
+                                InputType:    alerting.InputTypeText,
+                                Placeholder:  "https://matrix.example.org/_matrix/client/r0/rooms",
+                                PropertyName: "url",
+                                Required:     true,
+                        },
+                        {
+                                Label:        "Room ID",
+                                Element:      alerting.ElementTypeInput,
+                                InputType:    alerting.InputTypeText,
+                                Placeholder:  "Matrix Room ID",
+                                PropertyName: "roomid",
+                                Required:     true,
+                        },
+                        {
+                                Label:        "Token",
+                                Element:      alerting.ElementTypeInput,
+                                InputType:    alerting.InputTypeText,
+                                Placeholder:  "Authentication Token",
+                                PropertyName: "token",
+                                Required:     true,
+                        },
+                        {
+                                Label:        "Message Type",
+                                Element:      alerting.ElementTypeSelect,
+                                SelectOptions: []alerting.SelectOption{
+                                        {
+                                                Value: "m.notice",
+                                                Label: "m.notice",
+                                        },
+                                        {
+                                                Value: "m.text",
+                                                Label: "m.text",
+                                        },
+                                },
+                                PropertyName: "msgtype",
+                        },
+		},
+
+	})
+
+}
+
+func NewMatrixNotifier(model *models.AlertNotification) (alerting.Notifier, error) {
+	url := model.Settings.Get("url").MustString()
+	if url == "" {
+		return nil, alerting.ValidationError{Reason: "Could not find url property in settings"}
+	}
+	roomid := model.Settings.Get("roomid").MustString()
+	if roomid == "" {
+		return nil, alerting.ValidationError{Reason: "Could not find roomid property in settings"}
+	}
+	token := model.Settings.Get("token").MustString()
+	if token == "" {
+		return nil, alerting.ValidationError{Reason: "Could not find token property in settings"}
+	}
+
+	return &MatrixNotifier{
+		NotifierBase: NewNotifierBase(model),
+		URL:          url,
+		RoomID:       roomid,
+		Token:        token,
+		MsgType:      model.Settings.Get("msgtype").MustString("m.notice"),
+		log:          log.New("alerting.notifier.matrix"),
+	}, nil
+}
+
+type MatrixNotifier struct {
+	NotifierBase
+	URL        string
+	RoomID     string
+	Token      string
+	MsgType    string
+	log        log.Logger
+}
+
+//func (this *MatrixNotifier) ShouldNotify(context *alerting.EvalContext) bool {
+//	return defaultShouldNotify(context)
+//}
+
+func (this *MatrixNotifier) Notify(evalContext *alerting.EvalContext) error {
+	this.log.Info("Sending Matrix notify message")
+
+	bodyJSON := simplejson.New()
+
+	message := evalContext.GetNotificationTitle()
+	message += " " + evalContext.Rule.Message
+
+	ruleURL, err := evalContext.GetRuleURL()
+	if err == nil {
+		message += " " + ruleURL
+	}
+
+	bodyJSON.Set("msgtype", this.MsgType)
+	bodyJSON.Set("body", message)
+
+	body, _ := bodyJSON.MarshalJSON()
+	matrixURL := fmt.Sprintf("%s/%s/send/m.room.message?access_token=%s", this.URL, this.RoomID, this.Token)
+	cmd := &models.SendWebhookSync{Url: matrixURL, Body: string(body)}
+
+	if err := bus.DispatchCtx(evalContext.Ctx, cmd); err != nil {
+		this.log.Error("Failed to send Matrix notify message", "error", err, "webhook", this.Name)
+		return err
+	}
+
+	return nil
+}
