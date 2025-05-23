import {sampleAppName, sampleMaintainer} from "./config";
import {goTo, Pages} from "./tools";

export class AppStorePage {
    constructor() {
        goTo(Pages.AppStore)
    }

    downloadSampleApp() {
        cy.get("#show-unofficial-apps-checkbox").click()
        cy.get('#search-bar').type(sampleAppName)
        cy.get('#search-button').click()
        cy.get('#install-button').click()
        return this
    }

    downloadSampleAppsOldVersion() {
        cy.get("#show-unofficial-apps-checkbox").click()
        cy.get('#search-bar').type(sampleAppName)
        cy.get('#search-button').click()
        cy.get('table tr').contains(sampleAppName).parent('tr').within(() => {
            cy.get('td.latest-version-column').within(() => {
                cy.get('.v-select').click()
            })
        })
        cy.get('.v-list-item').contains('1.0').click()
        cy.get('#install-button').click()
        return this
    }

    assertSearchWorks() {
        cy.reload()
        cy.get('body').should('contain', 'Please search for apps.')
        cy.get('#search-bar').type('sampleapp')
        cy.get("#show-unofficial-apps-checkbox").click()
        cy.get('#search-button').click()

        cy.get('table tr').contains(sampleAppName).parent('tr').within(() => {
            cy.get('.maintainer-column').should('contain.text', sampleMaintainer)
            cy.get('.name-column').should('contain.text', sampleAppName)
            cy.get('.latest-version-column').should('contain.text', '2.0')
        })
        return this
    }
}