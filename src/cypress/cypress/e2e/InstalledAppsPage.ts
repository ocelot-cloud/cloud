import {
    cookie,
    ocelotCookie,
    ocelotUrl,
    frontendRootDomain,
    sampleAppName,
    setCookie,
    ocelotDbName,
    ocelotDbMaintainer
} from "./config";
import {clickConfirmationButton, goTo, Pages} from "./tools";

export class InstalledAppsPage {
    constructor() {
        cy.intercept('POST', '/api/apps/list').as('appsList')
        goTo(Pages.InstalledApps)
        cy.get('#app-list').should('be.visible')
    }

    assertAppStatus(expectedStatus: string) {
        this.getSampleAppRow().find('.status-column').should('have.text', expectedStatus).then((statusElement) => {
            if (expectedStatus === "Available") {
                cy.wrap(statusElement).should('have.class', 'text-success')
            } else if (expectedStatus === "Uninitialized") {
                cy.wrap(statusElement).should('have.class', 'text-warning')
            } else {
                throw new Error(`Unexpected status: ${expectedStatus}`)
            }
        });
        return this;
    }

    startApp() {
        this.getSampleAppRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#start-app-button').click()
        return this
    }

    assertAppWebsiteContent(expectedContent: string) {
        cy.request({
            url: '/api/secret',
        }).then((secretResponse) => {
            if (secretResponse.status !== 200 || !secretResponse.body) {
                throw new Error("Failed to fetch secret from backend");
            }
            const secret = String(secretResponse.body);
            let appUrl = `http://${sampleAppName}.localhost:8080/api`;
            cy.request({
                url: `${appUrl}?secret=${secret}`,
            }).then((res) => {
                expect(res.body).to.include(expectedContent);
            });
        });
        return this;
    }

    stopApp() {
        this.getSampleAppRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#stop-app-button').click()
        return this
    }

    deleteApp() {
        this.getSampleAppRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#prune-app-button').click()
        clickConfirmationButton()
        return this
    }

    private getSampleAppRow() {
        return cy.get('#app-list').contains('td', sampleAppName).parent('tr');
    }

    private getOcelotDbRow() {
        return cy.get('#app-list').contains('td', ocelotDbName).parent('tr');
    }

    assertColumnPresenceForAdmins() {
        cy.get('#app-list').find('tbody tr').should('have.length', 2)
        cy.get('#app-list').should('contain', 'Maintainer')
        cy.get('#app-list').should('contain', 'Name')
        cy.get('#app-list').should('contain', 'Current Version')
        cy.get('#app-list').should('contain', 'Link')
        cy.get('#app-list').should('contain', 'Operation')
        cy.get('#app-list').should('contain', 'Status')
        return this
    }

    assertColumnPresenceForUsers() {
        cy.get('#app-list').should('not.contain', 'Maintainer')
        cy.get('#app-list').should('contain', 'Name')
        cy.get('#app-list').should('not.contain', 'Current Version')
        cy.get('#app-list').should('contain', 'Link')
        cy.get('#app-list').should('not.contain', 'Operation')
        cy.get('#app-list').should('not.contain', 'Status')
        cy.get('#app-list').find('tbody tr').should('have.length', 1)
        return this
    }

    pruneAppIfExists() {
        cy.get('body').then(($body) => {
            if ($body.text().includes('sampleapp')) {
                this.deleteApp()
            }
        });
        return this;
    }

    assertVersion(versionName: string) {
        cy.get('#app-list').find('.version-name-cell').should('contain.text', versionName)
        return this
    }

    updateApp() {
        this.getSampleAppRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#update-app-button').click()
        return this
    }

    createSampleAppBackup() {
        this.getSampleAppRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#backup-app-button').click()
        return this
    }

    createOcelotDbBackup() {
        this.getOcelotDbRow().within(() => {
            cy.get('#operations-dropdown').click()
        })
        cy.get('#backup-app-button').click()
        return this
    }

    assertOcelotDbAppProperties() {
        cy.get('#app-list tbody tr').should('have.length', 1);

        cy.get('#app-list tbody tr')
            .should('contain', ocelotDbMaintainer)
            .and('contain', ocelotDbName)
            .and('contain', '17.2')

        cy.get('#app-list tbody tr')
            .contains('td', ocelotDbName)
            .parent()
            .find('#open-button')
            .should('be.disabled');

        cy.get('#app-list tbody tr')
            .contains('td', ocelotDbName)
            .parent()
            .find('#operations-dropdown')
            .click();

        cy.get('body')
            .should('not.contain', "Start")
            .and('not.contain', 'Stop')
            .and('not.contain', 'Update')
            .and('not.contain', 'Delete')
        cy.get('body').should('contain', "Backup")

        cy.get('#app-list tbody tr')
            .contains('td', ocelotDbName)
            .parent()
            .find('#operations-dropdown')
            .click()

        cy.get('body').should('contain.text', "Available")
        return this
    }

    assertSampleAppNotPresent() {
        cy.get('body').should('not.contain.text', sampleAppName)
    }
}

export function loginAsAdmin() {
    if (cookie == "") {
        cy.visit(ocelotUrl);
        cy.url().should('include', '/login')
        loginAsUser('admin')
    } else {
        cy.setCookie(ocelotCookie, cookie)
        cy.visit(ocelotUrl)
        cy.url().should('equal', ocelotUrl + "/")
    }
}

export function logout() {
    goTo(Pages.InstalledApps)
    cy.get('#logout').click()
    setCookie("")
    cy.clearCookies()
    cy.location('pathname').should('equal', '/login');
}

export var userPassword = "password"

export function setUserPassword(password: string) {
    userPassword = password
}

export function loginAsUser(user: string) {
    return cy.get('#username-field')
        .type(user)
        .get('#password-field')
        .type(userPassword)
        .get('#login-button')
        .click()
        .location('pathname')
        .should((path) => {
            expect(path).not.to.equal('/login');
        })
        .getCookie(ocelotCookie)
        .should('exist')
        .then((retrievedCookie) => {
            setCookie(retrievedCookie.value)
        });
}
