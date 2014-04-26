var Game = {
    display: null,
 
    init: function() {
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
        document.body.appendChild(this.canvas);

        this.lastMoveTime = 0;
        this.loadTestMode = false;

        this.mapUpdateQueue = new Queue();
        this.drawQueue = new Queue();
        this.sendMoveQueue = new Queue();
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
        initReq.open("get", "/static/wsaddr", false);
        initReq.send();
        var wsaddr = (initReq.responseText).trim();

        var wsocket = new WebSocket("ws://"+wsaddr+"/ws");
        this.ws = wsocket;
        this.ws.onopen = function() {
            var initMsg = JSON.stringify({"type":"init"});
            wsocket.send(initMsg);
        };
        this.ws.onmessage = function(event) {
            var jsonObj = JSON.parse(event.data);
            //var jsonObj = eval("("+event.data + ")");
            console.log(event.data);
            if (jsonObj.type === "init") {
                Game.uuid = jsonObj.uuid; 
                Game.renderDisplay(jsonObj.data);
                Game.initialized = true;
            }
            //if (Game.initialized && (jsonObj.type === "update")) {
            if (jsonObj.type === "update") {
                Game.mapUpdateQueue.enqueue(jsonObj.data);
            }
        };

        window.addEventListener("keydown", Game.handleKeyboardInput);
        this.canvas.addEventListener("mouseup", Game.handleMouseEvent);

        var animFrame = null;

        //navigator.platform === "Win32"
        //if (typeof WebSocket != 'undefined') { /*supported*/ } 
        if ((navigator.userAgent.indexOf("Firefox") > -1) &&
            (navigator.platform == "Win32")) {
            animFrame = null;
        } else {
            animFrame = window.requestAnimationFrame ||
                window.webkitRequestAnimationFrame ||
                window.mozRequestAnimationFrame    ||
                window.oRequestAnimationFrame      ||
                window.msRequestAnimationFrame     ||
                null ;
        }

        var updateGame = function() {
            var currentTime = (new Date).getTime();
            if (! Game.drawQueue.isEmpty()) {
                var mapToDraw = Game.drawQueue.dequeue();
                Game.display.drawEntire(mapToDraw);
            } else if (! Game.mapUpdateQueue.isEmpty()) {
                var updateObj = Game.mapUpdateQueue.dequeue();
                Game.renderDisplay(updateObj);
            } else if (((currentTime - Game.lastMoveTime) > Game.requestInterval) && (! Game.sendMoveQueue.isEmpty())) {
                var actions = '';
                while ( ! Game.sendMoveQueue.isEmpty()) {
                    actions = actions + Game.sendMoveQueue.dequeue();
                }
                var jsonObj = {"type":"mv", "data":actions};
                var data = JSON.stringify(jsonObj);
                console.log("sending "+data);
                Game.lastMoveTime = currentTime;
                Game.ws.send(actions);
                //Game.ws.send(data);
            } else if (Game.loadTestMode && 
                       ((currentTime - Game.lastMoveTime) > Game.requestInterval * 9) && 
                       Game.sendMoveQueue.isEmpty()) 
            {
                Game.sendMove("nneessww");
            }
            //console.log("poll: "+currentTime);
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
    }
};

Game.coord = function(x, y) {
    return Game.coordCache[y][x];
}

Game.sendMove = function(data) {
    Game.sendMoveQueue.enqueue(data);
    //var data = JSON.stringify(jsonObj);
    //Game.ws.send(data);
};

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
          "fg":"#FFF419",
          "bg":"#000"
        },
    "%":{ "disp":"%",
          "fg":"#FFF",
          "bg":"#000"
        }
};

Game.draw = function(aMapToDraw) {
    var mapToDraw = Game.drawQueue.dequeue();
    Game.display.drawEntire(mapToDraw);
    // Draw the player 
    Game.display.draw(Game.centerx, Game.centery, "@", "#1283B2", "#000");
}

Game.commitCell = function (drawMap,i, j, cellValue) {
    if (Game.previousCell(i,j) != cellValue.charCodeAt(0)) {
        var key = Game.coord(i,j);
        var anArray = Game.arrayCache.dequeue();
        anArray[0] = i;
        anArray[1] = j;
        if (cellValue.length === 1) {
            var scheme = Game.displayScheme[cellValue]; 
            anArray[2] = scheme.disp;
            anArray[3] = scheme.fg;
            anArray[4] = scheme.bg;
        } else {
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
    for (var j = 0; j < Game.dheight; j++) {
        for (var i = 0; i < Game.dwidth; i++) {
            var cellValue = Game.bufferCell(i,j);
            var key = Game.coord(i,j);
            var entity = Game.entities[key];
            if (entity) {
                cellValue = entity.symbol; 
            }
            Game.commitCell(drawMap,i,j,cellValue);
        }
    }
    // ensure you draw the player differently
    drawMap[Game.coord(Game.centerx,Game.centery)] = [Game.centerx,Game.centery,"@","#1283B2", "#000"];
    Game.drawQueue.enqueue(drawMap);
};
 
Game.renderDisplay = function(updateObj) {
    if (updateObj.entities) {
        Game.entities = updateObj.entities; 
    }
    //var location = updateObj.location;
    //document.getElementById("locationDisp").innerHTML = "ROTCS - location: "+location[0]+","+location[1];
    if (updateObj.maptype === "basic") {
        for (var j = 0; j < Game.dheight; j++) {
            for (var i = 0; i < Game.dwidth; i++) {
                var cellValue = updateObj.map[j].charAt(i);
                Game.setBufferCell(i, j, cellValue);
            }
        }
        Game.commitDisplay();
    } else if (updateObj.maptype === "line") {
        var moveRecord = updateObj.moveRecord;
        var lines = updateObj.map;
        for (var i = 0; i < lines.length; i++) {
            var move = moveRecord[i];
            var line = lines[i];
            if (move == "n") { 
                Game.scrollMapNorth(line); 
            } else if (move == "s") { 
                Game.scrollMapSouth(line); 
            } else if (move == "w") { 
                Game.scrollMapWest(line); 
            } else if (move == "e") { 
                Game.scrollMapEast(line); 
            }
        }
        Game.commitDisplay();
    } else if (updateObj.maptype === "entity") {
        Game.commitDisplay();
    }
}

Game.handleKeyboardInput = function (e) {

    var code = e.keyCode; 

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

    Game.sendMove(action);

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
    var arrays_equal = function(a,b) { return !(a<b || b<a); };
    var moveToLetter = function(aMove) {
        if (arrays_equal(aMove,[-1,0])) { return "w"; }
        if (arrays_equal(aMove,[1,0]))  { return "e"; }
        if (arrays_equal(aMove,[0,-1])) { return "n"; }
        if (arrays_equal(aMove,[0,1]))  { return "s"; }
    };

    var coords = Game.display.eventToPosition(e);
    //console.log(coords);
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

Game.scrollMapNorth = function(newLine) {
    Game.yboffset = (Game.yboffset - 1 + Game.dheight) % Game.dheight;
    for (var x = 0; x < Game.dwidth; x++) {
        var cellValue = newLine.charAt(x);
        Game.setBufferCell(x, 0, cellValue);
    }
};

Game.scrollMapSouth = function(newLine) {
    Game.yboffset = (Game.yboffset + 1 + Game.dheight) % Game.dheight;
    for (var x = 0; x < Game.dwidth; x++) {
        var cellValue = newLine.charAt(x);
        Game.setBufferCell(x, (Game.dheight - 1), cellValue);
    }
};

Game.scrollMapWest = function(newLine) {
    Game.xboffset = (Game.xboffset - 1 + Game.dwidth) % Game.dwidth;
    for (var y = 0; y < Game.dheight; y++) {
        var cellValue = newLine.charAt(y);
        Game.setBufferCell(0, y, cellValue);
    }
};

Game.scrollMapEast = function(newLine) {
    Game.xboffset = (Game.xboffset + 1 + Game.dwidth) % Game.dwidth;
    for (var y = 0; y < Game.dheight; y++) {
        var cellValue = newLine.charAt(y);
        Game.setBufferCell((Game.dwidth - 1), y, cellValue);
    }
};

