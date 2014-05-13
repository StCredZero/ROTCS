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
        initReq.open("get", "/wsaddr", false);
        initReq.send();
        var wsaddr = (initReq.responseText).trim();

        var wsocket = new WebSocket("ws://"+wsaddr+"/ws");
        this.ws = wsocket;
        /*this.ws.onopen = function() {
            var initMsg = JSON.stringify({"type":"init"});
            wsocket.send(initMsg);
        };*/
        this.ws.onmessage = function(event) {
            var jsonObj = JSON.parse(event.data);
            //var jsonObj = eval("("+event.data + ")");
            //console.log(event.data);
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
                //var jsonObj = {"type":"mv", "data":actions};
                //var data = JSON.stringify(jsonObj);
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

Game.preMove = function(newLoc) {
    if (!Game.entityUnsafeAt(newLoc)) {
        Game.oldLocation = Game.location;
        Game.scrollTo(newLoc);
        Game.commitDisplay();
    }
}

Game.sendMove = function(data) {
    console.log("sending: ", data, Game.location);
    if ((!Game.oldLocation) && Game.location) {
        if ((data == "n") && Game.walkableAt(39,11)) {
            Game.preMove([Game.location[0], Game.location[1] - 1]);
        } else if ((data == "s") && Game.walkableAt(39,13)) {
            Game.preMove([Game.location[0], Game.location[1] + 1]);
        } else if ((data == "e") && Game.walkableAt(40,12)) {
            Game.preMove([Game.location[0] + 1, Game.location[1]]);
        } else if ((data == "w") && Game.walkableAt(38,12)) {
            Game.preMove([Game.location[0] - 1, Game.location[1]]);
        }
    }
    console.log("location now: ", Game.location);
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
            anArray[2] = scheme.disp;
            anArray[3] = scheme.fg;
            anArray[4] = scheme.bg;
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
    drawMap[Game.coord(Game.centerx,Game.centery)] = [Game.centerx,Game.centery,"@","#1283B2", "#000"];
    Game.drawQueue.enqueue(drawMap);
};
 
Game.renderDisplay = function(updateObj) {
    if (updateObj.entities) {
        Game.entities = updateObj.entities; 
    }

    if (updateObj.maptype === "basic") {
        Game.oldLocation = null;
        var loc = updateObj.location;
        if (loc) {
            Game.scrollTo(loc);
        }
        for (var j = 0; j < Game.dheight; j++) {
            for (var i = 0; i < Game.dwidth; i++) {
                var cellValue = updateObj.map[j].charAt(i);
                Game.setBufferCell(i, j, cellValue);
            }
        }
        Game.commitDisplay();
    } else if (updateObj.maptype === "line") {
        if (Game.oldLocation) {
            if (((Game.location)[0] != (updateObj.location)[0]) ||
                ((Game.location)[1] != (updateObj.location)[1])) {
                Game.scrollTo(Game.oldLocation);
            }
        } 
        Game.oldLocation = null;
        Game.scrollTo(updateObj.location);
        Game.drawLine(updateObj.start, updateObj.orientation, updateObj.line);
        Game.commitDisplay();
    } else if (updateObj.maptype === "entity") {
        if (Game.oldLocation && updateObj.collided) {
            var loc = updateObj.location;
            if (loc) {
                Game.scrollTo(loc);
            }
            Game.oldLocation = null;
        }
        Game.commitDisplay();
    }
    if (Game.location) {
        document.getElementById("locationDisp").innerHTML = "ROTCS - location: "+Game.location[0]+","+Game.location[1];
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

    } 
    Game.location = newloc;
    //console.log("scrollto-end: ", Game.location);
}

Game.drawLine = function(start, orientation, line) {
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

Game.scrollMapX = function(vec) {
    Game.xboffset = (Game.xboffset + vec + Game.dwidth) % Game.dwidth;
};

Game.scrollMapY = function(vec) {
    Game.yboffset = (Game.yboffset + vec + Game.dheight) % Game.dheight;
};


