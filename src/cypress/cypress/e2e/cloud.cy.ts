import {
  InstalledAppsPage,
  loginAsAdmin, loginAsUser, logout, setUserPassword, userPassword
} from "./InstalledAppsPage";
import {AppStorePage} from "./AppStorePage";
import {goTo, Pages} from "./tools";
import {ocelotUrl, sampleAppName} from "./config";
import {UsersPage} from "./UsersPage";
import {BackupsPage} from "./BackupsPage";
import {SettingsPage} from "./SettingsPage";
import {
  assertAppStorePageSearchInputValidation, assertChangePasswordInputValidation,
  assertLoginPageInputValidation,
  assertUserPageInputValidation
} from "./InputValidation";

function cleanup() {
  cy.request({
    method: 'POST',
    url: 'http://localhost:8080/api/wipe',
  }).then((response) => {
    expect(response.status).to.eq(200);
  });
}

function setup() {
  cleanup()
  loginAsAdmin()
}

describe('template spec', () => {

  before(() => {
    setup()
  });

  it('check dns-01 challenge', () => {
    setup()
    new SettingsPage()
        .setAndSaveHostValue("sample.com")
        .doDnsChallenge()
  })

  it('user changes password', () => {
    setup()
    new UsersPage()
        .createUser()
    logout()
    loginAsUser('user')
    cy.get("#change-password").click()
    cy.location('pathname').should('eq', '/change-password')
    cy.get("#password-field").type('newpassword')
    cy.get("#change-password-button").click()

    logout()
    setUserPassword('newpassword')
    loginAsUser('user')
    cy.location('pathname').should('eq', '/')
    logout()
    setUserPassword('password')

    loginAsAdmin()
    new UsersPage()
        .deleteUser()
        .shouldUserExist(false)
  })

  it('check user creation and deletion', () => {
    setup()
    let usersPage = new UsersPage()
    usersPage
        .shouldUserExist(false)
        .shouldUserNameInputBeEmpty(true)
        .shouldUserPasswordInputBeEmpty(true)

        .createUser()
        .shouldUserExist(true)
        .shouldUserNameInputBeEmpty(true)
        .shouldUserPasswordInputBeEmpty(true)

    cy.get('#username').type('user')
    cy.get('#password').type('password')
    Cypress.Promise.try(() => cy.get('#submit').click())
    usersPage
        .shouldUserNameInputBeEmpty(false)
        .shouldUserPasswordInputBeEmpty(false)

    usersPage
        .shouldUserExist(true)
        .deleteUser()
        .shouldUserExist(false)
  })

  it('check maintenance settings', () => {
    setup()
    new SettingsPage()
      .assertEnableAutoUpdates(true)
      .assertEnableAutoBackups(true)
      .assertMaintenanceHour(4)
      .setCustomMaintenanceSettings()
    new SettingsPage()
      .assertEnableAutoUpdates(false)
      .assertEnableAutoBackups(false)
      .assertMaintenanceHour(7)
  })

  it('assert apps search feature', () => {
    setup()
    new AppStorePage()
        .assertSearchWorks()
  })

  it('check frontend-side input validation', () => {
    cleanup()
    assertLoginPageInputValidation()
    setup()
    assertUserPageInputValidation()
    assertAppStorePageSearchInputValidation()
    assertChangePasswordInputValidation()
  })

  it('check remote backup', () => {
    setup()
    new SettingsPage()
        .addSampleSshSettings()
        .enableRemoteBackups()
        .saveRemoteBackupsSettings()
    new AppStorePage()
        .downloadSampleApp()
    new InstalledAppsPage()
        .createSampleAppBackup()
    new BackupsPage()
        .assertSampleAppBackupIsPresent("2.0", "manual-backup")
        .deleteSampleAppBackup()
        .assertNoBackupsListed()
        .selectRemoteRepo()
        .assertSampleAppBackupIsPresent("2.0", "manual-backup")
  })

  it('check unofficial apps search feature', () => {
    setup()
    new AppStorePage()
    cy.get("#search-bar").type(sampleAppName)
    cy.get("#search-button").click()
    cy.get("body").should('not.contain', sampleAppName)

    cy.get("#show-unofficial-apps-checkbox").click()
    cy.get("#search-button").click()
    cy.get("body").should('contain', sampleAppName)
  })

  it('check enabling remote backup enables remote backup listing', () => {
    setup()
    new BackupsPage()
        .shouldOnlyContainLocalRepoType()
    new SettingsPage()
        .addSampleSshSettings()
        .enableRemoteBackups()
        .saveRemoteBackupsSettings()
    new BackupsPage()
        .shouldContainBothRepoTypes()
  })

  it('check remote backup settings', () => {
    setup()
    new SettingsPage()
        .shouldRemoteBackupsBeEnabled(false)
        .assertEmptySshSettings()
        .enableRemoteBackups()
        .shouldRemoteBackupsBeEnabled(true)
        .addSampleSshSettings()
        .saveRemoteBackupsSettings()
    cy.reload()
    new SettingsPage()
        .readAndAssertSshSettings()
        .assertSshKnownHostsAndTestConnection()
  })

  it('check manual backup', () => {
    setup()
    new AppStorePage()
        .downloadSampleApp()
    new InstalledAppsPage()
        .createSampleAppBackup()
    new BackupsPage()
        .assertSampleAppBackupIsPresent("2.0", "manual-backup")
        .deleteSampleAppBackup()
        .assertNoBackupsListed()
    new InstalledAppsPage()
        .pruneAppIfExists()
  })

  it('check updates and pre-update backup', () => {
    setup()
    new AppStorePage()
        .downloadSampleAppsOldVersion()
    new BackupsPage()
        .assertNoBackupsListed()
    new SettingsPage()
        .setAndSaveHostValue("localhost")
    new InstalledAppsPage()
        .startApp()
        .assertVersion("1.0")
        .assertAppWebsiteContent("this is version 1.0")
        .updateApp()
        .assertVersion("2.0")
        .assertAppWebsiteContent("this is version 2.0")
    new BackupsPage()
        .assertSampleAppBackupIsPresent("1.0", "auto-backup")
        .restoreSampleAppBackup()
    new InstalledAppsPage()
        .assertVersion("1.0")
        .assertAppWebsiteContent("this is version 1.0")
    new BackupsPage()
        .deleteSampleAppBackup()
        .assertNoBackupsListed()
    new InstalledAppsPage()
        .deleteApp()
  })

  it('check ocelotdb app', () => {
    setup()
    new UsersPage()
        .createUser()
        .shouldUserExist(true)
    new InstalledAppsPage()
        .assertOcelotDbAppProperties()
        .createOcelotDbBackup()
    new UsersPage()
        .deleteUser()
        .shouldUserExist(false)
    new BackupsPage()
        .restoreOcelotDbBackup()
    new UsersPage()
        .shouldUserExist(true)
        .deleteUserIfExists()
    new BackupsPage()
        .deleteOcelotDbBackup()
  })

  it('check user access to apps', () => {
    setup()
    new AppStorePage()
        .downloadSampleApp()
    new InstalledAppsPage()
        .assertColumnPresenceForAdmins()
        .startApp()
    new UsersPage()
        .assertAdminUser()
        .createUser()
    new SettingsPage()
        .setAndSaveHostValue("localhost")

    logout()
    loginAsUser('user')
    new InstalledAppsPage()
        .assertColumnPresenceForUsers()
        .assertAppWebsiteContent('this is version 2.0')

    logout()
    loginAsAdmin()
    new InstalledAppsPage()
        .deleteApp()
    new UsersPage()
        .deleteUser()
  });

  it('settings page checks', () => {
    setup()
    new SettingsPage()
        .setAndSaveHostValue("localhost")
    cy.reload()
    new SettingsPage()
        .assertHostValue("localhost")
        .setAndSaveHostValue("localhost2")
    cy.reload()
    new SettingsPage()
        .assertHostValue("localhost2")
  });

  it('check redirects', () => {
    setup()
    cy.contains('Logged in as: admin')
    goTo(Pages.AppStore)
    goTo(Pages.InstalledApps)
    goTo(Pages.Users)
    goTo(Pages.Backups)
    goTo(Pages.Settings)
  });

  it('get app content', () => {
    setup()
    new InstalledAppsPage()
        .pruneAppIfExists()
    new AppStorePage()
        .downloadSampleApp()
    new SettingsPage()
        .setAndSaveHostValue("localhost")

    new InstalledAppsPage()
        .stopApp()
        .assertAppStatus('Uninitialized')
        .startApp()
        .assertAppStatus('Available')
        .assertAppWebsiteContent('this is version 2.0')
        .stopApp()
        .assertAppStatus('Uninitialized')
        .deleteApp()
  });

  it('download app from app store', () => {
    setup()
    new InstalledAppsPage()
        .pruneAppIfExists()
    new AppStorePage()
        .downloadSampleApp()
    new SettingsPage()
        .setAndSaveHostValue("localhost")
    new InstalledAppsPage()
        .startApp()
        .assertAppStatus('Available')
        .assertAppWebsiteContent('this is version 2.0')
        .deleteApp()
        .assertSampleAppNotPresent()
  })

  it('logout', () => {
    setup()
    logout()
    cy.visit(ocelotUrl)
    cy.location('pathname').should('equal', '/login')
  });
});
