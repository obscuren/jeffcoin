import QtQuick 2.0
import QtQuick.Controls 1.0;
import QtQuick.Layouts 1.0;
import QtQuick.Dialogs 1.0;
import QtQuick.Window 2.1;
import QtQuick.Controls.Styles 1.1

ApplicationWindow {
	id: root
	title: "United Jeff Coin NC"

	width: 360
	height: 140

	RowLayout {
		anchors {
			left: parent.left
			leftMargin: 5
			top: parent.top
			topMargin: 5
		}

		TextField {
			placeholderText: "Address";
			width: 150;
		}

		TextField {
			placeholderText: "XXX";
			width: 50;
		}

		Button {
			text: "Send"
			onClicked: {
			}
		}
	}

	statusBar: StatusBar {
		height: 30
		RowLayout {
			Button {
				objectName: "miningButton"
				onClicked: {
					if(jc.isMining) {
						jc.stopMiner()
						this.text = "Mine"
					} else {
						jc.startMiner()
						this.text = "Stop mining"
					}
				}
				text: "Mine"
			}
			
			Label {
				objectName: "balanceLabel"
				text: "UJC: " + jc.balance()
				y: 2
				anchors {
					top: parent.top
					topMargin: 5
				}
			}
		}
	}
}
