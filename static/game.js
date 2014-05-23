var Game = {
    init: function(term) {
        var dwidth = 79;  this.dwidth = dwidth;
        var dheight = 25; this.dheight = dheight;
        this.centerx = 39;
        this.centery = 12;
        this.display = ADisplay.init(79,25);

        this.displayNode = document.getElementById('displayArea');
        this.displayNode.appendChild(this.display.canvas);
        this.displayNode.tabIndex = 1;

        this.lastMoveTimestamp = 0;

        this.loadTestMode = false;
        this.moveKeyDown = false;

        this.sendQueue = new Queue();
        this.initialized = false;

        this.requestInterval = (1000.0 / 8.0);

        var initReq = new XMLHttpRequest();
        initReq.open("get", "/wsaddr", false);
        initReq.send();
        var wsaddr = (initReq.responseText).trim();

        var wsocket = new WebSocket(wsaddr);
        this.ws = wsocket;
        /*this.ws.onopen = function() {
            var initMsg = JSON.stringify({"type":"init"});
            wsocket.send(initMsg);
        };*/
        this.ws.onmessage = function(event) {
            var jsonObj = JSON.parse(event.data);
            if (jsonObj.type === "init") {
                Game.uuid = jsonObj.uuid; 
                if (jsonObj.approved) {
                    Game.initialized = true;
                } else {
                    Game.showMessage("Server full. Try again later.");
                    Game.showMessage("Pop:" + jsonObj.pop + " Load:" + jsonObj.load);
                }
            }
            if (Game.initialized && (jsonObj.type === "update")) {
                if (jsonObj.messages) {
                    var messages = jsonObj.messages
                    for (var i = 0; i < messages.length; i++) {
                        if (messages[i].length > 0) {
                            Game.showMessage(messages[i]);
                        }
                    }
                }
                Game.display.mapUpdateQueue.enqueue(jsonObj);
            }
            if (jsonObj.type === "message") {
                Game.showMessage(jsonObj.data);
            }
        };

        window.addEventListener("keydown", Game.handleKeyboardInput);
        window.addEventListener("keyup", Game.handleKeyboardUp);
        this.display.canvas.addEventListener("mouseup", Game.handleMouseEvent);

        var animFrame = null;


        animFrame = window.requestAnimationFrame ||
            window.webkitRequestAnimationFrame ||
            window.mozRequestAnimationFrame    ||
            window.oRequestAnimationFrame      ||
            window.msRequestAnimationFrame     ||
            null ;

        var updateGame = function() {
            if (Game.sendQueue.isEmpty()) {
                Game.display.tick()
            } else {
                var actions = null;
                for (actions = Game.sendQueue.dequeue(); actions; actions = Game.sendQueue.dequeue()) {
                    Game.ws.send([(new Date).getTime(),actions].join(":"));
                }
            } 
        };

        if ( animFrame !== null ) {
            var recursiveAnim = function() {
                updateGame();
                animFrame(recursiveAnim);
            };
            // start the mainloop
            animFrame( recursiveAnim );
        } else {
            //setInterval( updateGame, 1000.0 / 24.0 );
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

        this.term = term
        if (this.term) {
            this.term.setGame(this);
        } 
        this.health = 0;
        this.pop = 0;
        this.load = 0;
    }
};

Game.focusID = function() {
    return Game.hasFocus;
}

Game.setFocus = function(value) {
    if (!(value === Game.hasFocus)) {
        var currentTime = (new Date).getTime();
        if ((!Game.lastFocusTime) || (currentTime - Game.lastFocusTime > 100.0)) {
            console.log("game setting focus: ", value);
            Game.lastFocusTime = currentTime;
            Game.hasFocus = value;
            if (Game.hasFocus === "game") {
                Game.displayNode.borderColor = "green";
                Game.displayNode.focus();
            } else {
                Game.displayNode.borderColor = "#000000";
                Game.displayNode.blur();
            }
            Game.term.setFocus(value); 
        }
    }
}

Game.sendMove = function(data) {
    Game.display.preMove(data);
    Game.sendQueue.enqueue("mv:" + data);
};

Game.sendMessage = function(data) {
    Game.sendQueue.enqueue("ch:" + data);
    Game.showMessage("You say: '" + data + "'");
};

Game.showMessage = function(message) {
    if (Game.term) {
        Game.term.output(message+"<br>");
    }
}

Game.handleKeyboardInput = function (e) {

    var code = e.keyCode; 

    if (code == 13 && (Game.hasFocus === "game")) {
        Game.setFocus("term");
        return;
    }
    if (code == 27) {
        if (Game.hasFocus === "game") {
            Game.setFocus("term");
            return;
        } else if (Game.hasFocus === "term") {
            Game.setFocus("game");
        }
    }
    if (! (Game.hasFocus === "game")) { return; }

    if (code == 85) {
        // keyCode for "u"
        Game.loadTestMode = ! Game.loadTestMode;
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
    Game.sendMove(action);

};

Game.handleKeyboardUp = function (e) {
    if (! (Game.hasFocus === "game")) { return; }
};

Game.handleMouseEvent = function (e) {
    Game.setFocus("game");
    var arrays_equal = function(a,b) { return !(a<b || b<a); };
    var moveToLetter = function(aMove) {
        if (arrays_equal(aMove,[-1,0])) { return "w"; }
        if (arrays_equal(aMove,[1,0]))  { return "e"; }
        if (arrays_equal(aMove,[0,-1])) { return "n"; }
        if (arrays_equal(aMove,[0,1]))  { return "s"; }
    };

    var coords = Game.display.eventToPosition(e);
    //console.log(coords);
    if (!Game.display.walkableAt(coords[0],coords[1])) { return; }

    var pathCoords = Game.display.findPath(coords[0],coords[1]);
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
    Game.sendMove(actions);
    //console.log(moves.join(""));
};
