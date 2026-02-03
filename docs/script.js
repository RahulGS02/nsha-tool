// OS Detection
function detectOS() {
    const userAgent = window.navigator.userAgent;
    const platform = window.navigator.platform;
    const macosPlatforms = ['Macintosh', 'MacIntel', 'MacPPC', 'Mac68K'];
    const windowsPlatforms = ['Win32', 'Win64', 'Windows', 'WinCE'];
    
    if (macosPlatforms.indexOf(platform) !== -1) {
        return 'macOS';
    } else if (windowsPlatforms.indexOf(platform) !== -1) {
        return 'Windows';
    } else if (/Linux/.test(platform)) {
        return 'Linux';
    }
    return 'Unknown';
}

// Update download button based on OS
function updateDownloadButton() {
    const os = detectOS();
    const osNameSpan = document.getElementById('os-name');
    const downloadBtn = document.getElementById('download-primary');
    
    if (osNameSpan) {
        osNameSpan.textContent = os;
    }
    
    // Set download link based on OS
    if (downloadBtn) {
        let platform = 'linux-amd64';
        if (os === 'Windows') platform = 'windows-amd64';
        else if (os === 'macOS') platform = 'darwin-amd64';
        
        downloadBtn.onclick = () => {
            trackDownload(platform);
            window.location.href = `https://github.com/RahulGS02/nsha-tool/releases/latest/download/nsha-${platform}${os === 'Windows' ? '.exe' : ''}`;
        };
    }
}

// Track downloads (Google Analytics)
function trackDownload(platform) {
    if (typeof gtag !== 'undefined') {
        gtag('event', 'download', {
            'event_category': 'Downloads',
            'event_label': platform,
            'value': 1
        });
    }
    console.log('Download tracked:', platform);
}

// Fetch GitHub stats
async function fetchGitHubStats() {
    try {
        const response = await fetch('https://api.github.com/repos/RahulGS02/nsha-tool');
        const data = await response.json();

        const stars = data.stargazers_count || 0;
        document.getElementById('github-stars').textContent = stars;
        document.getElementById('footer-stars').textContent = stars;

        // Fetch download count from releases
        const releasesResponse = await fetch('https://api.github.com/repos/RahulGS02/nsha-tool/releases');
        const releases = await releasesResponse.json();
        
        let totalDownloads = 0;
        releases.forEach(release => {
            release.assets.forEach(asset => {
                totalDownloads += asset.download_count || 0;
            });
        });
        
        document.getElementById('total-downloads').textContent = totalDownloads.toLocaleString();
        document.getElementById('footer-downloads').textContent = totalDownloads.toLocaleString();
    } catch (error) {
        console.error('Error fetching GitHub stats:', error);
        document.getElementById('github-stars').textContent = '0';
        document.getElementById('total-downloads').textContent = '0';
        document.getElementById('footer-stars').textContent = '0';
        document.getElementById('footer-downloads').textContent = '0';
    }
}

// Smooth scrolling for navigation links
function setupSmoothScrolling() {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            const href = this.getAttribute('href');
            if (href !== '#' && href.length > 1) {
                e.preventDefault();
                const target = document.querySelector(href);
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            }
        });
    });
}

// Installation tabs
function setupInstallationTabs() {
    const tabButtons = document.querySelectorAll('.tab-btn');
    
    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const tabName = button.getAttribute('data-tab');
            
            // Remove active class from all buttons and contents
            document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
            
            // Add active class to clicked button and corresponding content
            button.classList.add('active');
            document.getElementById(`${tabName}-tab`).classList.add('active');
        });
    });
}

// Download button tracking
function setupDownloadTracking() {
    document.querySelectorAll('.btn-download').forEach(button => {
        button.addEventListener('click', (e) => {
            const platform = button.getAttribute('data-platform');
            trackDownload(platform);
            button.href = `https://github.com/RahulGS02/nsha-tool/releases/latest/download/nsha-${platform}${platform.includes('windows') ? '.exe' : ''}`;
        });
    });
}

// Demo terminal simulation
const demoResponses = {
    'diagnose': [
        '[STEP 1] Diagnosing repository...',
        '[WARNING] Found 3 issue(s):',
        '  1. [null-sha] refs/tags/null-tag',
        '  2. [null-sha] refs/heads/broken-branch',
        '  3. [missing-commit] 0000000001',
        '[INFO] Run "nsha fix" to fix these issues'
    ],
    'fix --dry-run': [
        '[STEP 1] Diagnosing repository...',
        '[INFO] Found 3 issue(s) to fix',
        '[STEP 2] Creating backup...',
        '[SUCCESS] Backup created',
        '[INFO] [DRY RUN] Preview of changes:',
        '  - Fix null SHA in refs/tags/null-tag',
        '  - Fix null SHA in refs/heads/broken-branch',
        '  - Fix missing commit reference',
        '[INFO] Run without --dry-run to apply fixes'
    ],
    'fix --yes': [
        '[STEP 1] Diagnosing repository...',
        '[INFO] Found 3 issue(s) to fix',
        '[STEP 2] Creating backup...',
        '[SUCCESS] Backup created at ~/nsha/20240203-105946/backup',
        '[STEP 3] Fixing issues...',
        '[SUCCESS] Fixed null SHA in refs/tags/null-tag',
        '[SUCCESS] Fixed null SHA in refs/heads/broken-branch',
        '[SUCCESS] Fixed missing commit reference',
        '[STEP 4] Running garbage collection...',
        '[SUCCESS] Repository cleaned',
        '[STEP 5] Verifying fixes...',
        '[SUCCESS] All issues fixed! Repository is healthy.'
    ],
    'verify': [
        '[STEP 1] Verifying repository integrity...',
        '[SUCCESS] No issues found! Repository is healthy.'
    ],
    'help': [
        'NSHA - Null SHA Fixer',
        '',
        'Available Commands:',
        '  diagnose    Detect null SHA and broken tree issues',
        '  fix         Fix null SHA issues automatically',
        '  verify      Verify repository integrity',
        '',
        'Flags:',
        '  -r, --repo      Path to Git repository',
        '  -v, --verbose   Verbose output',
        '  --dry-run       Preview changes without applying',
        '  -y, --yes       Skip confirmation prompt',
        '',
        'Examples:',
        '  nsha diagnose',
        '  nsha fix --dry-run',
        '  nsha fix --yes',
        '  nsha verify'
    ]
};

function setupDemoTerminal() {
    const demoInput = document.getElementById('demo-input');
    const demoTerminalBody = document.getElementById('demo-terminal-body');

    if (!demoInput || !demoTerminalBody) return;

    function addLine(text, isCommand = false) {
        const line = document.createElement('div');
        line.className = isCommand ? 'terminal-line' : 'terminal-line output';
        line.textContent = text;

        // Insert before the input line
        const inputLine = demoTerminalBody.querySelector('.terminal-input-line');
        demoTerminalBody.insertBefore(line, inputLine);

        // Scroll to bottom
        demoTerminalBody.scrollTop = demoTerminalBody.scrollHeight;
    }

    function executeCommand(cmd) {
        const trimmedCmd = cmd.trim();

        // Add command to terminal
        addLine(`$ nsha ${trimmedCmd}`, true);

        // Get response
        const response = demoResponses[trimmedCmd] || [
            `[ERROR] Unknown command: ${trimmedCmd}`,
            'Type "help" to see available commands'
        ];

        // Add response lines with delay
        response.forEach((line, index) => {
            setTimeout(() => {
                addLine(line);
            }, index * 100);
        });

        // Clear input
        demoInput.value = '';
    }

    // Handle Enter key
    demoInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            const cmd = demoInput.value;
            if (cmd.trim()) {
                executeCommand(cmd);
            }
        }
    });

    // Handle quick command buttons
    document.querySelectorAll('.demo-cmd-btn').forEach(button => {
        button.addEventListener('click', () => {
            const cmd = button.getAttribute('data-cmd');
            executeCommand(cmd);
        });
    });
}

// Terminal animation for hero section
function animateHeroTerminal() {
    const terminalBody = document.querySelector('.hero-terminal .terminal-body');
    if (!terminalBody) return;

    const lines = terminalBody.querySelectorAll('.terminal-line');
    lines.forEach((line, index) => {
        line.style.opacity = '0';
        setTimeout(() => {
            line.style.transition = 'opacity 0.5s';
            line.style.opacity = '1';
        }, index * 300);
    });
}

// Initialize everything when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    updateDownloadButton();
    fetchGitHubStats();
    setupSmoothScrolling();
    setupInstallationTabs();
    setupDownloadTracking();
    setupDemoTerminal();
    animateHeroTerminal();

    // Refresh stats every 5 minutes
    setInterval(fetchGitHubStats, 5 * 60 * 1000);
});

// Handle mobile menu (if needed in future)
function setupMobileMenu() {
    // Add mobile menu toggle functionality here if needed
}

// Analytics tracking (if Google Analytics is loaded)
if (typeof gtag !== 'undefined') {
    // Track page view
    gtag('config', 'GA_MEASUREMENT_ID', {
        page_path: window.location.pathname
    });
}

