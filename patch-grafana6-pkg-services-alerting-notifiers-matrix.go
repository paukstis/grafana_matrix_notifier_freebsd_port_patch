--- /dev/null
+++ pkg/services/alerting/notifiers/matrix.go
@@ -0,0 +1,108 @@
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
+		Factory:     NewMatrixNotifier,
+		OptionsTemplate: `
+      <h3 class="page-heading">Matrix settings</h3>
+      <div class="gf-form">
+        <span class="gf-form-label width-10">Matrix room URL</span>
+        <input type="text" required class="gf-form-input" ng-model="ctrl.model.settings.url" placeholder="https://matrix.example.org/_matrix/client/r0/rooms"></input>
+      </div>
+      <div class="gf-form">
+        <span class="gf-form-label width-10">Room ID</span>
+        <input type="text" required class="gf-form-input" ng-model="ctrl.model.settings.roomid" placeholder="Matrix Room ID"></input>
+      </div>
+      <div class="gf-form">
+        <span class="gf-form-label width-10">Token</span>
+        <input type="text" required class="gf-form-input" ng-model="ctrl.model.settings.token" placeholder="Authentication Token"></input>
+      </div>
+      <div class="gf-form">
+        <span class="gf-form-label width-10">Message Type</span>
+        <div class="gf-form-select-wrapper width-10">
+          <select class="gf-form-input" ng-model="ctrl.model.settings.msgtype" ng-options="t for t in ['m.notice', 'm.text']"
+            ng-init="ctrl.model.settings.msgtype=ctrl.model.settings.msgtype||'m.notice'">
+          </select>
+        </div>
+      </div>
+    `,
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
--- /dev/null
+++ pkg/services/alerting/notifiers/matrix_test.go
@@ -0,0 +1,52 @@
+package notifiers
+
+import (
+	"testing"
+
+	"github.com/grafana/grafana/pkg/components/simplejson"
+	"github.com/grafana/grafana/pkg/models"
+	. "github.com/smartystreets/goconvey/convey"
+)
+
+func TestMatrixNotifier(t *testing.T) {
+	Convey("Matrix notifier tests", t, func() {
+
+		Convey("Parsing alert notification from settings", func() {
+			Convey("empty settings should return error", func() {
+				json := `{ }`
+
+				settingsJSON, _ := simplejson.NewJson([]byte(json))
+				model := &models.AlertNotification{
+					Name:     "matrix_testing",
+					Type:     "matrix",
+					Settings: settingsJSON,
+				}
+
+				_, err := NewMatrixNotifier(model)
+				So(err, ShouldNotBeNil)
+			})
+
+			Convey("from settings", func() {
+				json := `
+				{
+          "url": "http://google.com"
+				}`
+
+				settingsJSON, _ := simplejson.NewJson([]byte(json))
+				model := &models.AlertNotification{
+					Name:     "matrix_testing",
+					Type:     "matrix",
+					Settings: settingsJSON,
+				}
+
+				not, err := NewMatrixNotifier(model)
+				matrixNotifier := not.(*MatrixNotifier)
+
+				So(err, ShouldBeNil)
+				So(matrixNotifier.Name, ShouldEqual, "matrix_testing")
+				So(matrixNotifier.Type, ShouldEqual, "matrix")
+				So(matrixNotifier.Url, ShouldEqual, "http://google.com")
+			})
+		})
+	})
+}
