// Pauses or resumes media in the apps macOS lets us script. There is no public
// system-wide media API (MediaRemote is private and locked down since macOS
// 15.4), so each player family is handled on its own.
//
// argv[0]: 'pause' | 'resume'
// argv[1]: JSON array of {app, title} to resume (title is unused on macOS)
// Output:  JSON array of {app, title} that were actually paused or resumed
//
// Browsers need "Allow JavaScript from Apple Events" turned on (Chromium:
// View > Developer; Safari: Settings > Advanced > Show features for web
// developers, then Develop menu). Without it the tab calls throw and the
// browser is skipped.

const PLAYERS = ['Spotify', 'Music'];
const CHROMIUM = ['Google Chrome', 'Brave Browser', 'Microsoft Edge', 'Arc', 'Vivaldi', 'Chromium'];

// Marks the elements this tool paused, so resume never starts media the user
// paused themselves. dataset.headsUpPaused is the data-heads-up-paused attribute.
const PAUSE_JS = `(() => {
	let n = 0;
	document.querySelectorAll('video, audio').forEach(m => {
		if (!m.paused) { m.pause(); m.dataset.headsUpPaused = '1'; n++; }
	});
	return n;
})()`;

const RESUME_JS = `(() => {
	let n = 0;
	document.querySelectorAll('[data-heads-up-paused]').forEach(m => {
		delete m.dataset.headsUpPaused; m.play(); n++;
	});
	return n;
})()`;

// Application(name).running() checks without launching the app.
function isRunning(name) {
	try { return Application(name).running(); } catch (e) { return false; }
}

function eachTab(app, fn) {
	let n = 0;
	try {
		app.windows().forEach(w => w.tabs().forEach(t => {
			try { n += Number(fn(t)) || 0; } catch (e) {}
		}));
	} catch (e) {}
	return n;
}

function run(argv) {
	const action = argv[0];
	const targets = JSON.parse(argv[1] || '[]');
	const shouldTouch = name => action === 'pause' || targets.some(t => t.app === name);
	const js = action === 'pause' ? PAUSE_JS : RESUME_JS;
	const done = [];

	for (const name of PLAYERS) {
		if (!shouldTouch(name) || !isRunning(name)) continue;
		try {
			const app = Application(name);
			const state = app.playerState();
			if (action === 'pause' && state === 'playing') { app.pause(); done.push({app: name, title: ''}); }
			if (action === 'resume' && state === 'paused') { app.play(); done.push({app: name, title: ''}); }
		} catch (e) {}
	}

	for (const name of CHROMIUM) {
		if (!shouldTouch(name) || !isRunning(name)) continue;
		if (eachTab(Application(name), t => t.execute({javascript: js})) > 0) {
			done.push({app: name, title: ''});
		}
	}

	if (shouldTouch('Safari') && isRunning('Safari')) {
		const safari = Application('Safari');
		if (eachTab(safari, t => safari.doJavaScript(js, {in: t})) > 0) {
			done.push({app: 'Safari', title: ''});
		}
	}

	return JSON.stringify(done);
}
