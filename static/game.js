var Game = {
    display: null,
 
    init: function(term) {
        var dwidth = 79;  this.dwidth = dwidth;
        var dheight = 25; this.dheight = dheight;
        this.centerx = 39;
        this.centery = 12;
        this.display = new ROT.Display({
            "width":dwidth,
            "height":dheight,
            "fontFamily":"courier"
        });
        this.canvas = this.display.getContainer();
        this.displayNode = document.getElementById('displayArea');
        this.displayNode.appendChild(this.canvas);
        this.displayNode.tabIndex = 1;

        this.lastUpdateTimestamp = 0;
        this.lastMoveTimestamp = 0;
        this.lineStash = [];
        this.loadTestMode = false;
        this.moveKeyDown = false;

        this.mapUpdateQueue = new Queue();
        this.drawQueue = new Queue();
        this.sendQueue = new Queue();
        this.initialized = false;

        this.requestInterval = (1000.0 / 8.0);

        var initBuffer = function(anArray, cellFunc) {
            for (var j = 0; j < dheight; j++) {
                anArray[j] = [];
                for (var i = 0; i < dwidth; i++) {
                    anArray[j][i] = cellFunc(i,j); 
                }
            }
        }
        this.coordCache = [];
        initBuffer(this.coordCache, function(x,y){return x+","+y});
        this.display.setCoordCache(this.coordCache);
        this.drawBuffer = [];
        initBuffer(this.drawBuffer, function(x,y){ return " "; });
        this.xboffset = 0;
        this.yboffset = 0;
        this.previousBuffer = [];
        initBuffer(this.previousBuffer, function(x,y){ return 0; });
        this.arrayCache = new Queue();
        for (var n = 0; n < (2 * (dwidth * dheight)); n++) {
            this.arrayCache.enqueue(new Array());
        }

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
            //if (Game.initialized && (jsonObj.type === "update")) {
            if (Game.initialized && (jsonObj.type === "update")) {
                Game.mapUpdateQueue.enqueue(jsonObj);
            }
            if (jsonObj.type === "message") {
                Game.showMessage(jsonObj.data);
            }
        };

        window.addEventListener("keydown", Game.handleKeyboardInput);
        window.addEventListener("keyup", Game.handleKeyboardUp);
        this.canvas.addEventListener("mouseup", Game.handleMouseEvent);

        var animFrame = null;

        //navigator.platform === "Win32"
        //if (typeof WebSocket != 'undefined') { /*supported*/ } 
        //if ((navigator.userAgent.indexOf("Firefox") > -1) &&
        //    (navigator.platform == "Win32")) {
        //    animFrame = null;
        //} else {
            animFrame = window.requestAnimationFrame ||
                window.webkitRequestAnimationFrame ||
                window.mozRequestAnimationFrame    ||
                window.oRequestAnimationFrame      ||
                window.msRequestAnimationFrame     ||
                null ;
        //}

        var updateGame = function() {
            if (! Game.drawQueue.isEmpty()) {
                var mapToDraw = Game.drawQueue.dequeue();
                Game.display.drawEntire(mapToDraw);
            } else if (! Game.mapUpdateQueue.isEmpty()) {
                var updateObj = Game.mapUpdateQueue.dequeue();
                Game.renderDisplay(updateObj);
            } else if (! Game.sendQueue.isEmpty()) {
                var actions = null;
                for (actions = Game.sendQueue.dequeue(); actions; actions = Game.sendQueue.dequeue()) {
                    Game.ws.send([(new Date).getTime(),actions].join(":"));
                }
            } 
        };

        if ( animFrame !== null ) {
            //var mycanvas = this.canvas;
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

Game.coord = function(x, y) {
    return Game.coordCache[y][x];
}

Game.entityUnsafeAt = function(newLoc) {
    var x = newLoc[0]
    var y = newLoc[1]
    var k0 = [x,y-1].join(",")
    var k1 = [x,y+1].join(",")
    var k2 = [x+1,y].join(",")
    var k3 = [x-1,y].join(",")
    var k4 = [x,y].join(",")

    return Game.entities[k0] || Game.entities[k1] || Game.entities[k2] ||
        Game.entities[k3] || Game.entities[k4]
}
//Game.bufferCell(i,j)
Game.preMove = function(move) {
    var now = (new Date).getTime();
    if (now < Game.lastUpdateTimestamp + Game.requestInterval) {
        return
    }

    var newLoc = false
    var line = []
    if ((move == "n") && Game.walkableAt(39,11)) {
        newLoc = [Game.location[0], Game.location[1] - 1];
        if (!Game.entityUnsafeAt(newLoc)) {
            for (var i = 0; i < Game.dwidth; i++) {
                line.push(Game.bufferCell(i, Game.dheight - 1));
                Game.setBufferCell(i, Game.dheight - 1, Game.bufferCell(i, 0));
            }
            var stashStart = [Game.location[0] - 39, Game.location[1] + 12];
            Game.lineStash.push([stashStart, move, line.join("")]);
            Game.lastUpdateTimestamp = (new Date).getTime();
        }
    } else if ((move == "s") && Game.walkableAt(39,13)) {
        newLoc = [Game.location[0], Game.location[1] + 1];
        if (!Game.entityUnsafeAt(newLoc)) {
            for (var i = 0; i < Game.dwidth; i++) {
                line.push(Game.bufferCell(i, 0));
                Game.setBufferCell(i, 0, Game.bufferCell(i, Game.dheight - 1));
            }
            var stashStart = [Game.location[0] - 39, Game.location[1] - 12];
            Game.lineStash.push([stashStart, move, line.join("")]);
            Game.lastUpdateTimestamp = (new Date).getTime();
        }
    } else if ((move == "e") && Game.walkableAt(40,12)) {
        newLoc = [Game.location[0] + 1, Game.location[1]];
        if (!Game.entityUnsafeAt(newLoc)) {
            for (var j = 0; j < Game.dheight; j++) {
                line.push(Game.bufferCell(0, j));
                Game.setBufferCell(0, j, Game.bufferCell(Game.dwidth - 1, j));
            }
            var stashStart = [Game.location[0] - 39, Game.location[1] - 12];
            Game.lineStash.push([stashStart, move, line.join("")]);
            Game.lastUpdateTimestamp = (new Date).getTime();
        }
    } else if ((move == "w") && Game.walkableAt(38,12)) {
        newLoc = [Game.location[0] - 1, Game.location[1]];
        if (!Game.entityUnsafeAt(newLoc)) {
            for (var j = 0; j < Game.dheight; j++) {
                line.push(Game.bufferCell(Game.dwidth - 1, j));
                Game.setBufferCell(Game.dwidth - 1, j, Game.bufferCell(0, j));
            }
            var stashStart = [Game.location[0] + 39, Game.location[1] - 12];
            Game.lineStash.push([stashStart, move, line.join("")]);
            Game.lastUpdateTimestamp = (new Date).getTime();
        }
    }
    if (line.length > 0) {
        Game.oldLocation = Game.location;
        Game.scrollTo(newLoc);
        Game.lastMoveTimestamp = (new Date).getTime();
        Game.commitDisplay();
    }
}

Game.sendMove = function(data) {
    //Game.term.output("sending: "+data+"<br>");
    if ((!Game.oldLocation) && Game.location && Game.health > 0) {
        Game.preMove(data);
    }
    Game.sendQueue.enqueue("mv:" + data);
    //var data = JSON.stringify(jsonObj);
    //Game.ws.send(data);
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

Game.displayScheme = {
    ".":{ "disp":" ",
          "fg":"#FFF",
          "bg":"#000" 
        },
    " ":{ "disp":" ",
          "fg":"#000",
          "bg":"#B0B0B0"
        },
    "@":{ "disp":"@",
          "fg":"#004DFF",
          "bg":"#000"
        },
    "%":{ "disp":"%",
          "fg":"#FFF",
          "bg":"#000"
        },
    "+":{ "disp":"+",
          "fg":"#FFF",
          "bg":"#000"
        },
    "G":{ "disp":"G",
          "fg":"#004DFF",
          "bg":"#000"
        }
};

Game.draw = function(aMapToDraw) {
    var mapToDraw = Game.drawQueue.dequeue();
    Game.display.drawEntire(mapToDraw);
    // Draw the player 
    Game.display.draw(Game.centerx, Game.centery, "@", "#FFAA00", "#000");
}

Game.walkableAt = function (i,j) {
    return Game.previousCell(i,j) == (".".charCodeAt(0))
}

Game.commitCell = function (drawMap,i, j, cellValue) {
    if (Game.previousCell(i,j) != cellValue.charCodeAt(0)) {
        var key = Game.coord(i,j);
        var anArray = Game.arrayCache.dequeue();
        anArray[0] = i;
        anArray[1] = j;
        if (cellValue.length === 1) {
            var scheme = Game.displayScheme[cellValue]; 
            if (scheme) {
                anArray[2] = scheme.disp;
                anArray[3] = scheme.fg;
                anArray[4] = scheme.bg;
            } else {
                anArray[2] = cellValue;
                anArray[3] = "#FFF";
                anArray[4] = "#000";
            }
        } else if (cellValue.length > 1) {
            var dispChar = cellValue.substr(0,1);
            var scheme = Game.displayScheme[dispChar]; 
            var colorInfo = cellValue.substr(1);
            var colorArray = cellValue.split("#");
            if (colorArray.length === 1) {
                anArray[2] = scheme.disp;
                anArray[3] = colorArray[0];
                anArray[4] = scheme.bg;
            } else {
                anArray[2] = scheme.disp;
                anArray[3] = colorArray[0];
                anArray[4] = colorArray[1];
            }
        }
        //this.display.drawArray(i, j, anArray);
        drawMap[key] = anArray;
        Game.arrayCache.enqueue(anArray);
        Game.setPreviousCell(i,j,cellValue.charCodeAt(0));
    }
};

Game.commitDisplay = function() {
    var drawMap = {};
    var cornerx = Game.location[0] - Game.centerx;
    var cornery = Game.location[1] - Game.centery;
    for (var j = 0; j < Game.dheight; j++) {
        for (var i = 0; i < Game.dwidth; i++) {
            var cellValue = Game.bufferCell(i,j);
            var key = [(cornerx+i), (cornery+j)].join(","); //var key = Game.coord(i,j);
            var entity = Game.entities[key];
            if (entity) {
                cellValue = entity.symbol; 
            }
            Game.commitCell(drawMap,i,j,cellValue);
        }
    }
    // ensure you draw the player differently
    drawMap[Game.coord(Game.centerx,Game.centery)] = [Game.centerx,Game.centery,"@","#FFAA00", "#000"];
    Game.drawQueue.enqueue(drawMap);
};
 
Game.renderDisplay = function(updateObj) {
    if (updateObj.entities) { Game.entities = updateObj.entities; }
    if (updateObj.health) { Game.health = updateObj.health; }
    if (updateObj.pop) { Game.pop = updateObj.pop }
    if (updateObj.load) { Game.load = updateObj.load }
    if (updateObj.messages) {
        var messages = updateObj.messages
        for (var i = 0; i < messages.length; i++) {
            if (messages[i].length > 0) {
                Game.showMessage(messages[i]);
            }
        }
    }

    if (updateObj.maptype === "basic") {
        if (Game.lastUpdateTimestamp <= updateObj.timestamp) {
            if (Game.scrollTo(updateObj.location)) {
                for (var i = 0; i < Game.lineStash.length; i++) {
                    var stash = Game.lineStash[i];
                    Game.drawStash(stash[0], stash[1], stash[2]);
                }
            }
            Game.lastUpdateTimestamp = updateObj.timestamp;
            Game.oldLocation = null;
        }
        if (!Game.location) {
            Game.location = updateObj.location;
        }
        var cx = updateObj.location[0] - 39;
        var cy = updateObj.location[1] - 12;
        Game.drawBase64Map(cx, cy, 79, 25, updateObj.map);
        Game.commitDisplay();
    } else if (updateObj.maptype === "line") {
        if (Game.lastUpdateTimestamp <= updateObj.timestamp) {
            if (Game.scrollTo(updateObj.location)) {
                for (var i = 0; i < Game.lineStash.length; i++) {
                    var stash = Game.lineStash[i];
                    Game.drawStash(stash[0], stash[1], stash[2]);
                }
            }
            Game.lastUpdateTimestamp = updateObj.timestamp;
            Game.oldLocation = null;
        }
        Game.drawLine(updateObj.start, updateObj.orientation, updateObj.line);
        Game.commitDisplay();
    } else if (updateObj.maptype === "entity") {
        if (Game.lastUpdateTimestamp <= updateObj.timestamp) {
            if (Game.scrollTo(updateObj.location)) {
                for (var i = 0; i < Game.lineStash.length; i++) {
                    var stash = Game.lineStash[i];
                    Game.drawStash(stash[0], stash[1], stash[2]);
                }
            }
            Game.lastUpdateTimestamp = updateObj.timestamp;
            Game.oldLocation = null;
        }
        Game.commitDisplay();
    }
    if (Game.location) {
        document.getElementById("locationDisp").innerHTML = 
            ["Health:", Game.health, 
             " Location:" , Game.location[0],",",Game.location[1],
             " Users:", Game.pop,
             " Server Load:",Game.load].join("");
    }
    if (Game.lineStash.length > 0) {
        Game.lineStash.shift();
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

Game.mapAt = function(x, y) {
    return Game.bufferCell(x,y);
};

Game.findPath = function(x, y) {
    var passableCallback = function(x, y) {
        return (Game.mapAt(x,y) === ".");
    }
    var astar = new ROT.Path.AStar(Game.centerx, Game.centery, passableCallback, {topology:4});
    var path = [];
    var pathCallback = function(x1, y1) {
        path.push([x1, y1]);
    }
    astar.compute(x, y, pathCallback);
    return path;
}

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
    if (!Game.walkableAt(coords[0],coords[1])) { return; }

    var pathCoords = Game.findPath(coords[0],coords[1]);
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

Game.setLocationCell = function(x, y, cellValue) {
    var cornerx = Game.location[0] - Math.floor(Game.dwidth / 2)
    var cornery = Game.location[1] - Math.floor(Game.dheight / 2)
    var localx = x - cornerx;
    var localy = y - cornery;
    if (localx >= 0 && localy >= 0 && localx < Game.dwidth && localy < Game.dheight) {
        Game.setBufferCell(localx, localy, cellValue);
    }
}
 
Game.setBufferCell = function(x, y, cellValue) {
    var h = Game.dheight;
    var w = Game.dwidth;
    var xoffset = Game.xboffset;
    var yoffset = Game.yboffset;
    Game.drawBuffer[(y + yoffset + h) % h][(x + xoffset + w) % w] = cellValue;
}

Game.bufferCell = function(x, y) {
    return Game.drawBuffer[(y + Game.yboffset + Game.dheight) % Game.dheight][(x + Game.xboffset + Game.dwidth) % Game.dwidth];
}

Game.setPreviousCell = function(x, y, cellValue) {
    Game.previousBuffer[y][x] = cellValue;
}

Game.previousCell = function(x, y) {
    return Game.previousBuffer[y][x];
}

Game.sign = function(x) {
    if (x < 0) {
        return -1
    } else if (x > 0) {
        return 1
    } else {
        return 0
    }
}

Game.scrollTo = function(newloc) {
    //console.log("scrollto-begin: ", Game.location);
    if (Game.location) {
        var newx = newloc[0];
        var newy = newloc[1];
        var oldx = Game.location[0];
        var oldy = Game.location[1];
        var xvec = newx - oldx;
        var yvec = newy - oldy;
        //console.log("scrollto: ", xvec, yvec);
        Game.scrollMapX(xvec);
        Game.scrollMapY(yvec);
        Game.location = newloc;
        return (xvec !== 0 || yvec !== 0);
    } 
    Game.location = newloc;
    return false;
    //console.log("scrollto-end: ", Game.location);
}

Game.drawLine = function(start, orientation, line) {
    var x1 = start[0];
    var y1 = start[1];
    if (orientation == "n" || orientation == "s") {
        Game.drawBase64Map(x1, y1, Game.dwidth, 1, line);
    }
    if (orientation == "w" || orientation == "e") {
        Game.drawBase64Map(x1, y1, 1, Game.dheight, line);
    }
};

Game.drawStash = function(start, orientation, line) {
    var x1 = start[0];
    var y1 = start[1];
    if (orientation == "n" || orientation == "s") {
        for (var x = x1; x < (x1 + Game.dwidth); x++) {
            var cellValue = line.charAt(x - x1);
            Game.setLocationCell(x, y1, cellValue);
        }
    }
    if (orientation == "w" || orientation == "e") {
        for (var y = y1; y < (y1 + Game.dheight); y++) {
            var cellValue = line.charAt(y - y1);
            Game.setLocationCell(x1, y, cellValue);
        }
    }
};

Game.drawBase64Map = function(x0, y0, xsize, ysize, data) {
    var x = 0;
    var y = 0;
    for (var di = 0; di < data.length; di++) {
        var c = data.charAt(di);
        var v = Game.base64Table[c];
        for (var i = 0; i < 6; i++) {
            var cellValue = " ";
            
            if (v[i] == 1) {
                cellValue = ".";
            }
            if (x < xsize && y < ysize) {
                Game.setLocationCell(x + x0, y + y0, cellValue);
            }
            x += 1;
            if (x == xsize) {
                x = 0;
                y += 1;
            }
        }
    }
};

Game.scrollMapX = function(vec) {
    Game.xboffset = (Game.xboffset + vec + Game.dwidth) % Game.dwidth;
};

Game.scrollMapY = function(vec) {
    Game.yboffset = (Game.yboffset + vec + Game.dheight) % Game.dheight;
};

Game.base64Table = {
"A":[0,0,0,0,0,0],
"B":[1,0,0,0,0,0],
"C":[0,1,0,0,0,0],
"D":[1,1,0,0,0,0],
"E":[0,0,1,0,0,0],
"F":[1,0,1,0,0,0],
"G":[0,1,1,0,0,0],
"H":[1,1,1,0,0,0],
"I":[0,0,0,1,0,0],
"J":[1,0,0,1,0,0],
"K":[0,1,0,1,0,0],
"L":[1,1,0,1,0,0],
"M":[0,0,1,1,0,0],
"N":[1,0,1,1,0,0],
"O":[0,1,1,1,0,0],
"P":[1,1,1,1,0,0],
"Q":[0,0,0,0,1,0],
"R":[1,0,0,0,1,0],
"S":[0,1,0,0,1,0],
"T":[1,1,0,0,1,0],
"U":[0,0,1,0,1,0],
"V":[1,0,1,0,1,0],
"W":[0,1,1,0,1,0],
"X":[1,1,1,0,1,0],
"Y":[0,0,0,1,1,0],
"Z":[1,0,0,1,1,0],
"a":[0,1,0,1,1,0],
"b":[1,1,0,1,1,0],
"c":[0,0,1,1,1,0],
"d":[1,0,1,1,1,0],
"e":[0,1,1,1,1,0],
"f":[1,1,1,1,1,0],
"g":[0,0,0,0,0,1],
"h":[1,0,0,0,0,1],
"i":[0,1,0,0,0,1],
"j":[1,1,0,0,0,1],
"k":[0,0,1,0,0,1],
"l":[1,0,1,0,0,1],
"m":[0,1,1,0,0,1],
"n":[1,1,1,0,0,1],
"o":[0,0,0,1,0,1],
"p":[1,0,0,1,0,1],
"q":[0,1,0,1,0,1],
"r":[1,1,0,1,0,1],
"s":[0,0,1,1,0,1],
"t":[1,0,1,1,0,1],
"u":[0,1,1,1,0,1],
"v":[1,1,1,1,0,1],
"w":[0,0,0,0,1,1],
"x":[1,0,0,0,1,1],
"y":[0,1,0,0,1,1],
"z":[1,1,0,0,1,1],
"0":[0,0,1,0,1,1],
"1":[1,0,1,0,1,1],
"2":[0,1,1,0,1,1],
"3":[1,1,1,0,1,1],
"4":[0,0,0,1,1,1],
"5":[1,0,0,1,1,1],
"6":[0,1,0,1,1,1],
"7":[1,1,0,1,1,1],
"8":[0,0,1,1,1,1],
"9":[1,0,1,1,1,1],
"+":[0,1,1,1,1,1],
"/":[1,1,1,1,1,1]};
