var CreateGame = function(term) {

    var gameState_ = "START";

    var display_ = ADisplay.init(79,25);

    var displayNode_ = document.getElementById('displayArea');
    displayNode_.appendChild(display_.canvas);
    displayNode_.tabIndex = 1;

    var loadTestMode_ = false;
    var requestInterval_ = (1000.0 / 8.0);
    var lastMoveTime_ = 0;
    var moveKeyDown_ = null;

    var sendQueue_ = new Queue();

    var initReq = new XMLHttpRequest();
    initReq.open("get", "/wsaddr", false);
    initReq.send();

    var uuid_ = null;

    var generateInterval_ = function(k) {
        var maxInterval = (Math.pow(2, k) - 1) * 1000;
  
        if (maxInterval > 30*1000) {
            maxInterval = 30*1000; // If the generated interval is more than 30 seconds, truncate it down to 30 seconds.
        }
  
        // generate the interval to a random number between 0 and the maxInterval determined from above
        return Math.random() * maxInterval; 
    };

    var wsaddr = (initReq.responseText).trim();
    var wsocket_ = null;
    var reconnectAttempts_ = 1;

    var animFrame = null;

    animFrame = window.requestAnimationFrame ||
	window.webkitRequestAnimationFrame ||
	window.mozRequestAnimationFrame    ||
	window.oRequestAnimationFrame      ||
	window.msRequestAnimationFrame     ||
	null ;

    var sendImmediate_ = function(action) {
        wsocket_.send([(new Date).getTime(),action].join(":"));
    }

    var wss = [];

    var updateGame = function() {
	display_.tick()
        if (((lastMoveTime_ + requestInterval_) < (new Date).getTime()) && 
            sendQueue_.isEmpty() && moveKeyDown_) {
            sendMove_(moveKeyDown_);
        }
	if (!sendQueue_.isEmpty()) {
	    var actions = null;
	    for (actions = sendQueue_.dequeue(); actions; actions = sendQueue_.dequeue()) {
		sendImmediate_(actions);
	    }
	} 
    };

    if ( animFrame !== null ) {
	var recursiveAnim = function() {
	    updateGame();
	    animFrame(recursiveAnim);
	};
	animFrame( recursiveAnim );
    } else {
	var updateTimer = -1
	var step = 1000.0 / 32.0;
	var nextUpdate = function() {
	    updateTimer = setTimeout(function() {
		clearTimeout(updateTimer);    
		updateGame();    
		nextUpdate();
	    }, step);
	};
	// start the mainloop
	nextUpdate();
    }

    var term_ = term;
    var hasFocus_ = null;
    var lastFocusTime_ = 0;
    var setFocus_ = function(value) {
	if (!(value === hasFocus_)) {
	    var currentTime = (new Date).getTime();
	    if ((!lastFocusTime_) || (currentTime - lastFocusTime_ > 100.0)) {
		console.log("game setting focus: ", value);
		lastFocusTime_ = currentTime;
		hasFocus_ = value;
		if (hasFocus_ === "game") {
		    displayNode_.borderColor = "green";
		    displayNode_.focus();
		} else {
                    moveKeyDown_ = null;
                    sendQueue_ = new Queue();
		    displayNode_.borderColor = "#000000";
		    displayNode_.blur();
		}
		term_.setFocus(value); 
	    }
	}
    };

    var sendMove_ = function(data) {
	display_.preMove(data);
        lastMoveTime_ = (new Date).getTime();
	sendQueue_.enqueue("mv:" + data);
    };

    var sendMessage_ = function(data) {
	sendQueue_.enqueue("ch:" + data);
	showMessage_("You say: '" + data + "'");
    };

    var showMessage_ = function(message) {
	if (term_) {
	    term_.output(message+"<br>");
	}
    };

    var handleKeyboardInput_ = function (e) {

	var code = e.keyCode; 

	if (code == 13 && (hasFocus_ === "game")) {
	    e.preventDefault();
	    e.stopPropagation();
	    setFocus_("term");
	    return;
	}
	if (code == 27) {
	    e.preventDefault();
	    e.stopPropagation();
	    if (hasFocus_ === "game") {
		setFocus_("term");
		return;
	    } else if (hasFocus_ === "term") {
		setFocus_("game");
	    }
	}
	if (! (hasFocus_ === "game")) { return; }

	if (code == 85) {
	    // keyCode for "u"
	    e.preventDefault();
	    e.stopPropagation();
            loadTestMode_ = !loadTestMode_;
	    return;
	};

	var action = "0";

	if (code == 38) { action = "n"; }
	if (code == 40) { action = "s"; }
	if (code == 37) { action = "w"; }
	if (code == 39) { action = "e"; } 

	if (code === "0") { return; }

	e.preventDefault();
	e.stopPropagation();
        if (moveKeyDown_ === action) { return; }
        moveKeyDown_ = action;
	sendMove_(action);
    };

    var handleKeyboardUp_ = function (e) {
	if (! (hasFocus_ === "game")) { return; }
	var code = e.keyCode; 
        //console.log(code)

        // a for life activate
        if (code == 65) {
	    e.preventDefault();
	    e.stopPropagation();
            if (display_.lifeAllowed()) {
                sendQueue_.enqueue("al:0");
            } else {
                showMessage_("Life System Inactive. Try a different grid.");
            }
            return
        }

        // l for life pen
        if (code == 76) {
            sendQueue_.enqueue("li:0");
	    e.preventDefault();
	    e.stopPropagation();
            return
        }

	var action = "0";
	if (code == 38) { action = "n"; }
	if (code == 40) { action = "s"; }
	if (code == 37) { action = "w"; }
	if (code == 39) { action = "e"; } 
	if (code === "0") { return; }
        if (moveKeyDown_ === action) {
            moveKeyDown_ = null;
            sendQueue_ = new Queue();
        }
    };

    var handleMouseEvent_ = function (e) {
	setFocus_("game");
	var arrays_equal = function(a,b) { return !(a<b || b<a); };
	var moveToLetter = function(aMove) {
	    if (arrays_equal(aMove,[-1,0])) { return "w"; }
	    if (arrays_equal(aMove,[1,0]))  { return "e"; }
	    if (arrays_equal(aMove,[0,-1])) { return "n"; }
	    if (arrays_equal(aMove,[0,1]))  { return "s"; }
	};

	var coords = display_.eventToPosition(e);
	//console.log(coords);
	if (!display_.walkableAt(coords[0],coords[1])) { return; }

	var pathCoords = display_.findPath(coords[0],coords[1]);
	//console.log(pathCoords);
	var moves = [];
	if (pathCoords) {
	    var coord = pathCoords.pop();
	    while (pathCoords.length > 0) {
		var newc = pathCoords.pop();
		var move = [(newc[0]-coord[0]),(newc[1] - coord[1])]; 
		moves.push(moveToLetter(move));
		coord = newc;
	    }
	}
	var actions = moves.join("");
	sendMove_(actions);
	//console.log(moves.join(""));
    };

    var handleBlur_ = function(e) {
        display_.handleBlur();
        sendImmediate_("bl:1");
        $.blockUI({ message: "<h1>Display Paused</h1> <h3>Running in Background</h3>" }); 
    };
    var handleFocus_ = function(e) {
        display_.handleFocus();
        sendImmediate_("bl:0");
        $.unblockUI();
    };

    function addEvent(obj, evType, fn, isCapturing){
      if (isCapturing==null) isCapturing=false; 
      if (obj.addEventListener){
        // Firefox
        obj.addEventListener(evType, fn, isCapturing);
        return true;
      } else if (obj.attachEvent){
        // MSIE
        var r = obj.attachEvent('on'+evType, fn);
        return r;
      } else {
        return false;
      }
    }

    // register to the W3C Page Visibility API
    var hidden=null;
    var visibilityChange=null;
    if (typeof document.mozHidden !== "undefined") {
      hidden="mozHidden";
      visibilityChange="mozvisibilitychange";
    } else if (typeof document.msHidden !== "undefined") {
      hidden="msHidden";
      visibilityChange="msvisibilitychange";
    } else if (typeof document.webkitHidden!=="undefined") {
      hidden="webkitHidden";
      visibilityChange="webkitvisibilitychange";
    } else if (typeof document.hidden !=="hidden") {
      hidden="hidden";
      visibilityChange="visibilitychange";
    }
    if (hidden!=null && visibilityChange!=null) {
      addEvent(document, visibilityChange, function(event) {
        if (document[hidden]) {
            handleBlur_();
        } else {
            handleFocus_();
        }
      });
    };

    window.addEventListener("keydown", handleKeyboardInput_);
    window.addEventListener("keyup",   handleKeyboardUp_);
    window.addEventListener("blur",    handleBlur_);
    window.addEventListener("focus",   handleFocus_);

    display_.canvas.addEventListener("mouseup", handleMouseEvent_);


    var initWebSockets_ = function(){
       try {
            wsocket_ = new WebSocket(wsaddr);
        } catch (err) {
            return false;
        };

        wsocket_.onmessage = function(event) {
	    var jsonObj = JSON.parse(event.data);
	    if (jsonObj.type === "init") {
	        if (jsonObj.approved) {
                    if (gameState_ === "RECONNECT") {
                        sendImmediate_([(new Date()).getTime(),"reconnect",uuid_].join(":"));
                    } else {
	                uuid_ = jsonObj.uuid; 
                        reconnectAttempts_ = 1;
                        gameState_ = "INITIALIZED";
                    }
	        } else {
		    Game.showMessage("Server full. Try again later.");
		    Game.showMessage("Pop:" + jsonObj.pop + " Load:" + jsonObj.load);
	        }
	    }
	    if ((gameState_ === "RECONNECT") && (jsonObj.type === "re")) {
                 if (jsonObj.approved) {
	             if (uuid_ !== jsonObj.uuid) {
                         throw "UUIDs don't match!";
                     }; 
                     reconnectAttempts_ = 1;
                     gameState_ = "INITIALIZED";
	        }
            }
	    if ((gameState_ === "INITIALIZED") && (jsonObj.type === "update")) {
	        if (jsonObj.messages) {
		    var messages = jsonObj.messages
		    for (var i = 0; i < messages.length; i++) {
		        if (messages[i].length > 0) {
			    showMessage_(messages[i]);
		        }
		    }
	        }
	        display_.queueUpdate(jsonObj);
	    }
	    if (jsonObj.type === "message") {
	        showMessage_(jsonObj.data);
	    }
        };

        wsocket_.onclose = function(event) {
            console.log("websocket closed");
            gameState_ = "RECONNECT";

            var time = generateInterval_(reconnectAttempts_);
    
            setTimeout(function () {
                // We've tried to reconnect so increment the attempts by 1
                reconnectAttempts_++;
                
                // Connection has closed so try to reconnect every 10
                // seconds.
                console.log("websocket reconnect");
                initWebSockets_(); 
            }, time);
        };

        return true;
    };
    initWebSockets_();

    return {
	focusID: (function() {return hasFocus_}),
        sendMessage: sendMessage_,
	setFocus: setFocus_
    }
    
};

