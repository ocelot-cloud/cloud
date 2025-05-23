import {clickConfirmationButton, goTo, Pages} from "./tools";
import {ocelotDbName, sampleAppName} from "./config";

function deleteBackup(appName: string, expectedVersionString: string) {
    cy.get('#app-selector').click()
    cy.get('.v-list-item').contains(appName).click()
    cy.get('table tr').should('have.length', 2);
    cy.get('table tr').contains(expectedVersionString).parents('tr').within(() => {
        cy.root().find('#actions-button').click()
    });
    cy.get('#delete-action').click()
    clickConfirmationButton()
    cy.reload()
    return this
}

export class BackupsPage {
    constructor() {
        goTo(Pages.Backups)
    }

    assertNoBackupsListed() {
        cy.get('#app-selector').click()
        cy.get('#app-selector').should('not.contain.text', '0') // ensure the selectedAppId was reset as well
        cy.get('body').should('not.contain.text', 'sampleapp')
        cy.get('body').should('contain.text', 'No data available')
        return this
    }

    assertSampleAppBackupIsPresent(expectedVersion: string, expectedDescription: string) {
        cy.get('#app-selector').click()
        cy.get('.v-list-item').contains('sampleapp').click()
        cy.get('table tr').should('have.length', 2)

        cy.get('table tr').contains('.0').parents('tr').within(() => {
            cy.root().should('contain.text', expectedVersion)
            cy.get('td').eq(2).invoke('text').then((dateText) => {
                const actualTime = new Date(dateText.trim());
                const lowerBound = new Date('2021/01/01');
                const upperBound = new Date('2021/01/01');
                lowerBound.setHours(lowerBound.getHours() - 48);
                upperBound.setHours(upperBound.getHours() + 48);
                expect(actualTime).to.be.within(lowerBound, upperBound);
            });
            cy.root().should('contain.text', expectedDescription)
        })
        return this
    }

    restoreSampleAppBackup() {
        cy.get('#app-selector').click()
        cy.get('.v-list-item').contains('sampleapp').click()
        cy.get('table tr').should('have.length', 2)
        cy.get('table tr').contains('.0').parents('tr').within(() => {
            cy.root().find('#actions-button').click()
        });
        cy.get('#restore-action').click()
        clickConfirmationButton()
        return this
    }

    restoreOcelotDbBackup() {
        cy.get('#app-selector').click()
        cy.get('.v-list-item').contains(ocelotDbName).click()
        cy.get('table tr').should('have.length', 2)
        cy.get('table tr').contains('17.2').parents('tr').within(() => {
            cy.root().find('#actions-button').click()
        });
        cy.get('#restore-action').click()
        clickConfirmationButton()
        return this
    }

    deleteSampleAppBackup() {
        deleteBackup(sampleAppName, ".0")
        return this
    }

    deleteOcelotDbBackup() {
        deleteBackup(ocelotDbName, "17.2")
        return this
    }

    shouldOnlyContainLocalRepoType() {
        cy.get('#backup-repository-selector').click({force: true})
        cy.contains('.v-list-item', 'Local')
        cy.get('.v-list-item').should('not.contain', 'Remote')
    }

    shouldContainBothRepoTypes() {
        cy.get('#backup-repository-selector').click({force: true})
        cy.contains('.v-list-item', 'Local')
        cy.contains('.v-list-item', 'Remote')
    }

    selectRemoteRepo() {
        cy.get('#backup-repository-selector').click({force: true})
        cy.contains('.v-list-item', 'Remote').click({ force: true })
        return this
    }
}
