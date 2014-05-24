var ADisplay = {
    init: function(dw,dh) {
        var dwidth_ = dw; 
        var dheight_ = dh;
        var centerx_ = Math.floor(dwidth_ / 2);
        var centery_ = Math.floor(dheight_ / 2);
        var location_ = null;
        var oldLocation_ = null;
        
        var display_ = new ROT.Display({
            "width":dwidth_,
            "height":dheight_,
            "fontFamily":"courier"
        });
        var canvas_ = display_.getContainer();
        
        var lastUpdateTimestamp_ = 0;
        var lastMoveTimestamp_ = 0;
        var lineStash_ = [];
        
        var mapUpdateQueue_ = new Queue();
        var drawQueue_ = new Queue();
        var initialized_ = false;
        
        var requestInterval_ = (1000.0 / 8.0);
        
        var initBuffer_ = function(anArray, cellFunc) {
            for (var j = 0; j < dheight_; j++) {
                anArray[j] = [];
                for (var i = 0; i < dwidth_; i++) {
                    anArray[j][i] = cellFunc(i,j); 
                }
            }
        }
        var coordCache_ = [];
        initBuffer_(coordCache_, function(x,y){return x+","+y});
        display_.setCoordCache(coordCache_);
        var drawBuffer_ = [];
        initBuffer_(drawBuffer_, function(x,y){ return " "; });
        var xboffset_ = 0;
        var yboffset_ = 0;
        var previousBuffer_ = [];
        initBuffer_(previousBuffer_, function(x,y){ return 0; });
        var arrayCache_ = new Queue();
        for (var n = 0; n < (2 * (dwidth_ * dheight_)); n++) {
            arrayCache_.enqueue(new Array());
        }
        
        var tick_ = function() {
            if (! drawQueue_.isEmpty()) {
                var mapToDraw = drawQueue_.dequeue();
                display_.drawEntire(mapToDraw);
            } else if (! mapUpdateQueue_.isEmpty()) {
                var updateObj = mapUpdateQueue_.dequeue();
                renderDisplay_(updateObj);
            } 
        };
        
        var health_ = 0;
        var pop_ = 0;
        var load_ = 0;
        
        var coord_ = function(x, y) {
            return coordCache_[y][x];
        };
        
        var entityUnsafeAt_ = function(newLoc) {
            var x = newLoc[0]
            var y = newLoc[1]
            var k0 = [x,y-1].join(",")
            var k1 = [x,y+1].join(",")
            var k2 = [x+1,y].join(",")
            var k3 = [x-1,y].join(",")
            var k4 = [x,y].join(",")
            
            return entities_[k0] || entities_[k1] || entities_[k2] ||
                entities_[k3] || entities_[k4]
        };
        
        
        var preMove_ = function(move) {
            
            if (oldLocation_ || (!location_) || health_ <= 0) { return; }
            var now = (new Date).getTime();
            if (now < lastUpdateTimestamp_ + requestInterval_) {
                return
            }
            
            var newLoc = false
            var line = []
            if ((move == "n") && walkableAt_(39,11)) {
                newLoc = [location_[0], location_[1] - 1];
                if (!entityUnsafeAt_(newLoc)) {
                    for (var i = 0; i < dwidth_; i++) {
                        line.push(bufferCell_(i, dheight_ - 1));
                        setBufferCell_(i, dheight_ - 1, bufferCell_(i, 0));
                    }
                    var stashStart = [location_[0] - 39, location_[1] + 12];
                    lineStash_.push([stashStart, move, line.join("")]);
                    lastUpdateTimestamp_ = (new Date).getTime();
                }
            } else if ((move == "s") && walkableAt_(39,13)) {
                newLoc = [location_[0], location_[1] + 1];
                if (!entityUnsafeAt_(newLoc)) {
                    for (var i = 0; i < dwidth_; i++) {
                        line.push(bufferCell_(i, 0));
                        setBufferCell_(i, 0, bufferCell_(i, dheight_ - 1));
                    }
                    var stashStart = [location_[0] - 39, location_[1] - 12];
                    lineStash_.push([stashStart, move, line.join("")]);
                    lastUpdateTimestamp_ = (new Date).getTime();
                }
            } else if ((move == "e") && walkableAt_(40,12)) {
                newLoc = [location_[0] + 1, location_[1]];
                if (!entityUnsafeAt_(newLoc)) {
                    for (var j = 0; j < dheight_; j++) {
                        line.push(bufferCell_(0, j));
                        setBufferCell_(0, j, bufferCell_(dwidth_ - 1, j));
                    }
                    var stashStart = [location_[0] - 39, location_[1] - 12];
                    lineStash_.push([stashStart, move, line.join("")]);
                    lastUpdateTimestamp_ = (new Date).getTime();
                }
            } else if ((move == "w") && walkableAt_(38,12)) {
                newLoc = [location_[0] - 1, location_[1]];
                if (!entityUnsafeAt_(newLoc)) {
                    for (var j = 0; j < dheight_; j++) {
                        line.push(bufferCell_(dwidth_ - 1, j));
                        setBufferCell_(dwidth_ - 1, j, bufferCell_(0, j));
                    }
                    var stashStart = [location_[0] + 39, location_[1] - 12];
                    lineStash_.push([stashStart, move, line.join("")]);
                    lastUpdateTimestamp_ = (new Date).getTime();
                }
            }
            if (line.length > 0) {
                oldLocation_ = location_;
                scrollTo_(newLoc);
                lastMoveTimestamp_ = (new Date).getTime();
                commitDisplay_();
            }
        };
        
        var displayScheme_ = {
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
        
        var draw_ = function(aMapToDraw) {
            var mapToDraw = drawQueue_.dequeue();
            display_.drawEntire(mapToDraw);
            // Draw the player 
            display_.draw(centerx_, centery_, "@", "#FFAA00", "#000");
        }
        
        var walkableAt_ = function (i,j) {
            return previousCell_(i,j) == (".".charCodeAt(0))
        }
        
        var commitCell_ = function (drawMap,i, j, cellValue) {
            if (previousCell_(i,j) != cellValue.charCodeAt(0)) {
                var key = coord_(i,j);
                var anArray = arrayCache_.dequeue();
                anArray[0] = i;
                anArray[1] = j;
                if (cellValue.length === 1) {
                    var scheme = displayScheme_[cellValue]; 
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
                    var scheme = displayScheme_[dispChar]; 
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
                
                drawMap[key] = anArray;
                arrayCache_.enqueue(anArray);
                setPreviousCell_(i,j,cellValue.charCodeAt(0));
            }
        };
        
        var commitDisplay_ = function() {
            var drawMap = {};
            var cornerx = location_[0] - centerx_;
            var cornery = location_[1] - centery_;
            for (var j = 0; j < dheight_; j++) {
                for (var i = 0; i < dwidth_; i++) {
                    var cellValue = bufferCell_(i,j);
                    var key = [(cornerx+i), (cornery+j)].join(","); //var key = coord_(i,j);
                    var symbol = entities_[key];
                    if (symbol) {
                        cellValue = symbol;
                    }
                    commitCell_(drawMap,i,j,cellValue);                    
                }
            }
            // ensure you draw the player differently
            drawMap[coord_(centerx_,centery_)] = [centerx_,centery_,"@","#FFAA00", "#000"];
            drawQueue_.enqueue(drawMap);
        };
        
        var setEntities_ = function(entities, loc) {
            var px = loc[0];
            var py = loc[1];
            var entityMap = {};
            for (var i = 0; i < entities.length; i += 3) {
                var x0 = base91Table_[entities.charAt(i)];
                var y0 = base91Table_[entities.charAt(i+1)];
                var x = x0 + px - Math.floor(dwidth_/2);
                var y = y0 + py - Math.floor(dheight_/2);
                var key = [x,y].join(",");
                entityMap[key] = entities.charAt(i+2);
            }
            entities_ = entityMap;
        };

        var renderDisplay_ = function(updateObj) {
            if (updateObj.entities && updateObj.location) { 
                setEntities_(updateObj.entities, updateObj.location); 
            }
            if (updateObj.health) { health_ = updateObj.health; }
            if (updateObj.pop) { pop_ = updateObj.pop }
            if (updateObj.load) { load_ = updateObj.load }
            
            if (updateObj.maptype === "basic") {
                if (lastUpdateTimestamp_ <= updateObj.timestamp) {
                    if (scrollTo_(updateObj.location)) {
                        for (var i = 0; i < lineStash_.length; i++) {
                            var stash = lineStash_[i];
                            drawStash_(stash[0], stash[1], stash[2]);
                        }
                    }
                    lastUpdateTimestamp_ = updateObj.timestamp;
                    oldLocation_ = null;
                }
                if (!location_) {
                    location_ = updateObj.location;
                }
                var cx = updateObj.location[0] - 39;
                var cy = updateObj.location[1] - 12;
                drawBase64Map_(cx, cy, 79, 25, updateObj.map);
                commitDisplay_();
            } else if (updateObj.maptype === "line") {
                if (lastUpdateTimestamp_ <= updateObj.timestamp) {
                    if (scrollTo_(updateObj.location)) {
                        for (var i = 0; i < lineStash_.length; i++) {
                            var stash = lineStash_[i];
                            drawStash_(stash[0], stash[1], stash[2]);
                        }
                    }
                    lastUpdateTimestamp_ = updateObj.timestamp;
                    oldLocation_ = null;
                }
                drawLine_(updateObj.start, updateObj.orientation, updateObj.line);
                commitDisplay_();
            } else if (updateObj.maptype === "entity") {
                if (lastUpdateTimestamp_ <= updateObj.timestamp) {
                    if (scrollTo_(updateObj.location)) {
                        for (var i = 0; i < lineStash_.length; i++) {
                            var stash = lineStash_[i];
                            drawStash_(stash[0], stash[1], stash[2]);
                        }
                    }
                    lastUpdateTimestamp_ = updateObj.timestamp;
                    oldLocation_ = null;
                }
                commitDisplay_();
            }
            if (location_) {
                document.getElementById("locationDisp").innerHTML = 
                    ["Health:", health_, 
                     " Location:" , location_[0],",",location_[1],
                     " Users:", pop_,
                     " Server Load:",load_].join("");
            }
            if (lineStash_.length > 0) {
                lineStash_.shift();
            }
        };
        
        var mapAt_ = function(x, y) {
            return bufferCell_(x,y);
        };
        
        var findPath_ = function(x, y) {
            var passableCallback = function(x, y) {
                return (mapAt_(x,y) === ".");
            }
            var astar = new ROT.Path.AStar(centerx_, centery_, passableCallback, {topology:4});
            var path = [];
            var pathCallback = function(x1, y1) {
                path.push([x1, y1]);
            }
            astar.compute(x, y, pathCallback);
            return path;
        };
        
        var setLocationCell_ = function(x, y, cellValue) {
            var cornerx = location_[0] - Math.floor(dwidth_ / 2)
            var cornery = location_[1] - Math.floor(dheight_ / 2)
            var localx = x - cornerx;
            var localy = y - cornery;
            if (localx >= 0 && localy >= 0 && localx < dwidth_ && localy < dheight_) {
                setBufferCell_(localx, localy, cellValue);
            }
        };
        
        var setBufferCell_ = function(x, y, cellValue) {
            var h = dheight_;
            var w = dwidth_;
            var xoffset = xboffset_;
            var yoffset = yboffset_;
            drawBuffer_[(y + yoffset + h) % h][(x + xoffset + w) % w] = cellValue;
        };
        
        var bufferCell_ = function(x, y) {
            return drawBuffer_[(y + yboffset_ + dheight_) % dheight_][(x + xboffset_ + dwidth_) % dwidth_];
        };
        
        var setPreviousCell_ = function(x, y, cellValue) {
            previousBuffer_[y][x] = cellValue;
        };
        
        var previousCell_ = function(x, y) {
            return previousBuffer_[y][x];
        };

        var scrollTo_ = function(newloc) {
            //console.log("scrollto-begin: ", location_);
            if (location_) {
                var newx = newloc[0];
                var newy = newloc[1];
                var oldx = location_[0];
                var oldy = location_[1];
                var xvec = newx - oldx;
                var yvec = newy - oldy;
                //console.log("scrollto: ", xvec, yvec);
                scrollMapX_(xvec);
                scrollMapY_(yvec);
                location_ = newloc;
                return (xvec !== 0 || yvec !== 0);
            } 
            location_ = newloc;
            return false;
            //console.log("scrollto-end: ", location_);
        }
        
        var drawLine_ = function(start, orientation, line) {
            var x1 = start[0];
            var y1 = start[1];
            if (orientation == "n" || orientation == "s") {
                drawBase64Map_(x1, y1, dwidth_, 1, line);
            }
            if (orientation == "w" || orientation == "e") {
                drawBase64Map_(x1, y1, 1, dheight_, line);
            }
        };
        
        var drawStash_ = function(start, orientation, line) {
            var x1 = start[0];
            var y1 = start[1];
            if (orientation == "n" || orientation == "s") {
                for (var x = x1; x < (x1 + dwidth_); x++) {
                    var cellValue = line.charAt(x - x1);
                    setLocationCell_(x, y1, cellValue);
                }
            }
            if (orientation == "w" || orientation == "e") {
                for (var y = y1; y < (y1 + dheight_); y++) {
                    var cellValue = line.charAt(y - y1);
                    setLocationCell_(x1, y, cellValue);
                }
            }
        };

        var drawBase64Map_ = function(x0, y0, xsize, ysize, data) {
            var x = 0;
            var y = 0;
            for (var di = 0; di < data.length; di++) {
                var c = data.charAt(di);
                var v = base64Table_[c];
                for (var i = 0; i < 6; i++) {
                    var cellValue = " ";
                    
                    if (v[i] == 1) {
                        cellValue = ".";
                    }
                    if (x < xsize && y < ysize) {
                        setLocationCell_(x + x0, y + y0, cellValue);
                    }
                    x += 1;
                    if (x == xsize) {
                        x = 0;
                        y += 1;
                    }
                }
            }
        };

        var queueUpdate_ = function(mapUpdate) {
            mapUpdateQueue_.enqueue(mapUpdate);
        };

        var scrollMapX_ = function(vec) {
            xboffset_ = (xboffset_ + vec + dwidth_) % dwidth_;
        };

        var scrollMapY_ = function(vec) {
            yboffset_ = (yboffset_ + vec + dheight_) % dheight_;
        };

        var eventToPosition_ = function(e) {
            return display_.eventToPosition(e);
        };

        var base64Table_ = {
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

        var base91Table_ = {
            "A":0,
            "B":1,
            "C":2,
            "D":3,
            "E":4,
            "F":5,
            "G":6,
            "H":7,
            "I":8,
            "J":9,
            "K":10,
            "L":11,
            "M":12,
            "N":13,
            "O":14,
            "P":15,
            "Q":16,
            "R":17,
            "S":18,
            "T":19,
            "U":20,
            "V":21,
            "W":22,
            "X":23,
            "Y":24,
            "Z":25,
            "a":26,
            "b":27,
            "c":28,
            "d":29,
            "e":30,
            "f":31,
            "g":32,
            "h":33,
            "i":34,
            "j":35,
            "k":36,
            "l":37,
            "m":38,
            "n":39,
            "o":40,
            "p":41,
            "q":42,
            "r":43,
            "s":44,
            "t":45,
            "u":46,
            "v":47,
            "w":48,
            "x":49,
            "y":50,
            "z":51,
            "0":52,
            "1":53,
            "2":54,
            "3":55,
            "4":56,
            "5":57,
            "6":58,
            "7":59,
            "8":60,
            "9":61,
            "!":62,
            "#":63,
            "$":64,
            "%":65,
            "&":66,
            "(":67,
            ")":68,
            "*":69,
            "+":70,
            ",":71,
            ".":72,
            "/":73,
            ":":74,
            ";":75,
            "<":76,
            "=":77,
            ">":78,
            "?":79,
            "@":80,
            "[":81,
            "]":82,
            "^":83,
            "_":84,
            "`":85,
            "{":86,
            "|":87,
            "}":88,
            "~":89,
            "-":90};
        
        return {
            canvas: canvas_,
            eventToPosition: eventToPosition_,
            findPath: findPath_,
            preMove: preMove_,
            queueUpdate: queueUpdate_,
            rotDisp: display_,
            tick: tick_,
            walkableAt: walkableAt_
        }
    }
};



