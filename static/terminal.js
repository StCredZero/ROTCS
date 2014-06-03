/*
Copyright 2014 Peter Kwangjun Suk.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Derived from Html5 terminal by Eric Bidelman (ericbidelman@chromium.org)
http://www.htmlfivewow.com/demos/terminal/terminal.html
*/

var util = util || {};
util.toArray = function(list) {
  return Array.prototype.slice.call(list || [], 0);
};

// Cross-browser impl to get document's height.
util.getDocHeight = function() {
  var d = document;
  return Math.max(
      Math.max(d.body.scrollHeight, d.documentElement.scrollHeight),
      Math.max(d.body.offsetHeight, d.documentElement.offsetHeight),
      Math.max(d.body.clientHeight, d.documentElement.clientHeight)
  );
};



var Terminal = Terminal || function(containerId) {
  window.URL = window.URL || window.webkitURL;
  window.requestFileSystem = window.requestFileSystem ||
                             window.webkitRequestFileSystem;

  const VERSION_ = '0.1';
  const CMDS_ = [
    'clear', 'date', 'exit', 'help', 'login', 'mission', 'say', 'version', 'who'
  ];
  const THEMES_ = ['default', 'cream'];

  var hasFocus_ = "";
  var game_ = null;

  var fs_ = null;
  var cwd_ = null;
  var history_ = [];
  var histpos_ = 0;
  var histtemp_ = 0;

  var timer_ = null;
  var magicWord_ = null;

  var fsn_ = null;

  // Create terminal and cache DOM nodes;
  var container_ = document.getElementById(containerId);
  container_.insertAdjacentHTML('beforeEnd',
      ['<output></output>',
       '<div id="input-line" class="input-line">',
       '<div class="prompt">$&gt;</div><div class="cmdwrap"><input class="cmdline" value=" " autofocus /></div>',
       '</div>'].join(''));
  var cmdLine_ = container_.querySelector('#input-line .cmdline');
  var output_ = container_.querySelector('output');
  var interlace_ = document.querySelector('.interlace');

  output_.addEventListener('click', function(e) {
    var el = e.target;
    if (el.classList.contains('file') || el.classList.contains('folder')) {
      cmdLine_.value += ' ' + el.textContent;
    }
  }, false);

  function setFocus_(value) {
      if (!(value === hasFocus_)) {
          console.log("term setting focus: ", value);
          hasFocus_ = value;
          if (hasFocus_ === "term") {
              cmdLine_.focus();
          } else {
              cmdLine_.blur();
          }
          if ((value === "term") && game_) { game_.setFocus(value); }
      }
  }

  function focusID_() {
      return hasFocus_;
  }

  function setGame_(gameObj) {
    game_ = gameObj;
  }

  window.addEventListener('click', function(e) {
    //if (!document.body.classList.contains('offscreen')) {
      //cmdLine_.focus();
    //}
  }, false);

  // Always force text cursor to end of input line.
  cmdLine_.addEventListener('click', inputTextClick_, false);

  // Handle up/down key presses for shell history and enter for new command.
  cmdLine_.addEventListener('keydown', keyboardShortcutHandler_, false);
  cmdLine_.addEventListener('keyup', historyHandler_, false); // keyup needed for input blinker to appear at end of input.
  cmdLine_.addEventListener('keydown', processNewCommand_, false);

  /*window.addEventListener('beforeunload', function(e) {
    return "Don't leave me!";
  }, false);*/

  function inputTextClick_(e) {
    setFocus_("term");
    this.value = this.value;
  }

  function keyboardShortcutHandler_(e) {
    if (! (hasFocus_ === "term")) { return; }

    // Toggle CRT screen flicker.
    if ((e.ctrlKey || e.metaKey) && e.keyCode == 83) { // crtl+s
      container_.classList.toggle('flicker');
      output('<div>Screen flicker: ' +
             (container_.classList.contains('flicker') ? 'on' : 'off') +
             '</div>');
      e.preventDefault();
      e.stopPropagation();
    }
  }

  function selectFile_(el) {
    alert(el)
  }

  function historyHandler_(e) { // Tab needs to be keydown.
    if (! (hasFocus_ === "term")) { return; }

    if (history_.length) {
      if (e.keyCode == 38 || e.keyCode == 40) {
        if (history_[histpos_]) {
          history_[histpos_] = this.value;
        } else {
          histtemp_ = this.value;
        }
      }

      if (e.keyCode == 38) { // up
        histpos_--;
        if (histpos_ < 0) {
          histpos_ = 0;
        }
      } else if (e.keyCode == 40) { // down
        histpos_++;
        if (histpos_ > history_.length) {
          histpos_ = history_.length;
        }
      }

      if (e.keyCode == 38 || e.keyCode == 40) {
        this.value = history_[histpos_] ? history_[histpos_] : histtemp_;
        this.value = this.value; // Sets cursor to end of input.
      }
    }
  }

  function processNewCommand_(e) {
    if (! (hasFocus_ === "term")) { return; }

    if (e.keyCode == 9) { // Tab
      e.preventDefault();
    } else if (e.keyCode == 13) { // enter

      // Save shell history.
      if (this.value) {
        history_[history_.length] = this.value;
        histpos_ = history_.length;
      }

      // Duplicate current input and append to output section.
      var line = this.parentNode.parentNode.cloneNode(true);
      line.removeAttribute('id')
      line.classList.add('line');
      var input = line.querySelector('input.cmdline');
      input.autofocus = false;
      input.readOnly = true;
      output_.appendChild(line);

      var lineText = '';

      // Parse out command, args, and trim off whitespace.
      if (this.value && this.value.trim()) {
        lineText = this.value;
        var args = lineText.split(' ').filter(function(val, i) {
          return val;
        });
        var cmd = args[0].toLowerCase();
        args = args.splice(1); // Remove cmd from arg list.
      }

      switch (cmd) {
      case 'c':
      case 'clear':
          clear_(this);
          return;
      case 'date':
          output((new Date()).toLocaleString());
          break;
      case 'exit':
          output('command under construction<br>');
          break;
      case 'help':
          output('<div class="ls-files">' + CMDS_.join('<br>') + '</div>');
          output('<p>Toggle command mode using the ESC key.</p>');
          output('<p>Arrow keys/click to move</p>');
          output('<p>L=LifePen. A=Activate Life System</p>');
          break;
      case 'login':
          output('command under construction<br>');
          break;
      case 'mission':
          output('command under construction<br>');
          break;
      case 's':
      case 'say':
          if (game_) {
              output('say (chat) is disabled (temporarily)<br>');
              /*if (args.length == 0) {
                  output('you said nothing<br>');
              } else {
                  game_.sendMessage(args.join(' '));
              }*/
          }
          break;
      case 'version':
      case 'ver':
          output(VERSION_);
          break;
      case 'who':
          output('command is under construction<br>');
          break;
      default:
          if (cmd) {
              output('command not found');
          }
      };

      this.value = ''; // Clear/setup line for next input.
    }
  }

  function invalidOpForEntryType_(e, cmd, dest) {
    if (e.code == FileError.NOT_FOUND_ERR) {
      output(cmd + ': ' + dest + ': No such file or directory<br>');
    } else if (e.code == FileError.INVALID_STATE_ERR) {
      output(cmd + ': ' + dest + ': Not a directory<br>');
    } else if (e.code == FileError.INVALID_MODIFICATION_ERR) {
      output(cmd + ': ' + dest + ': File already exists<br>');
    } else {
      errorHandler_(e);
    }
  }

  function errorHandler_(e) {
    var msg = '';
    switch (e.code) {
      case FileError.QUOTA_EXCEEDED_ERR:
        msg = 'QUOTA_EXCEEDED_ERR';
        break;
      case FileError.NOT_FOUND_ERR:
        msg = 'NOT_FOUND_ERR';
        break;
      case FileError.SECURITY_ERR:
        msg = 'SECURITY_ERR';
        break;
      case FileError.INVALID_MODIFICATION_ERR:
        msg = 'INVALID_MODIFICATION_ERR';
        break;
      case FileError.INVALID_STATE_ERR:
        msg = 'INVALID_STATE_ERR';
        break;
      default:
        msg = 'Unknown Error';
        break;
    };
    output('<div>Error: ' + msg + '</div>');
  }

  function clear_(input) {
    output_.innerHTML = '';
    input.value = '';
    document.documentElement.style.height = '100%';
    interlace_.style.height = '100%';
  }

  function setTheme_(theme) {
    var currentUrl = document.location.pathname;

    if (!theme || theme == 'default') {
      //history.replaceState({}, '', currentUrl);
      localStorage.removeItem('theme');
      document.body.className = '';
      return;
    }

    if (theme) {
      document.body.classList.add(theme);
      localStorage.theme = theme;
      //history.replaceState({}, '', currentUrl + '#theme=' + theme);
    }
  }

  function output(html) {
    output_.insertAdjacentHTML('beforeEnd', html);
    //output_.scrollIntoView();
    cmdLine_.scrollIntoView();
  }

  return {
    initFS: function(persistent, size) {
        output('<div>Welcome to ' + document.title +
             '! (v' + VERSION_ + ')</div>');
        output((new Date()).toLocaleString());
        output('<p>Documentation: ESC to toggle move/command mode</p>');
        output('<p>        Arrow keys/click to move</p>');
        output('<p>        Command "help" for more</p>');
    },
    output: output,
    setTheme: setTheme_,
    getCmdLine: function() { return cmdLine_; },
    setFocus: setFocus_,
    setGame: setGame_,
    focusID: focusID_
  }
};

