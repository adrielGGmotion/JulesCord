from playwright.sync_api import sync_playwright

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        page.goto("http://localhost:5173/guilds/123456789/settings")
        page.wait_for_timeout(2000)
        page.screenshot(path="guild_settings_page.png")

        browser.close()

if __name__ == "__main__":
    run()
