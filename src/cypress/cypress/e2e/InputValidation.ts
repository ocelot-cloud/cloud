import {ocelotUrl} from "./config";
import {UsersPage} from "./UsersPage";
import {AppStorePage} from "./AppStorePage";

export function assertLoginPageInputValidation() {
    cy.visit(ocelotUrl)
    cy.url().should('include', '/login')
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#username-field').type('a')
    cy.get('#password-field').type('password')
    cy.get('#login-button').click()
    cy.get('body').should('contain.text', 'Invalid input')

    cy.reload()
    cy.visit(ocelotUrl)
    cy.url().should('include', '/login')
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#username-field').type('user')
    cy.get('#password-field').type('a')
    cy.get('#login-button').click()
    cy.get('body').should('contain.text', 'Invalid input')
}

export function assertUserPageInputValidation() {
    new UsersPage()
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#username').type('user')
    cy.get('#password').type('a')
    cy.get('#submit').click()
    cy.get('body').should('contain.text', 'Invalid input')

    cy.reload()
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#username').type('a')
    cy.get('#password').type('password')
    cy.get('#submit').click()
    cy.get('body').should('contain.text', 'Invalid input')
}

export function assertAppStorePageSearchInputValidation() {
    new AppStorePage()
    cy.get('#search-button').click()
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#search-bar').type('a!')
    cy.get('body').should('contain.text', 'Invalid input')
}

export function assertChangePasswordInputValidation() {
    cy.visit(ocelotUrl + "/change-password")
    cy.get('#password-field').type('a!')
    cy.get('body').should('not.contain.text', 'Invalid input')
    cy.get('#change-password-button').click()
    cy.get('body').should('contain.text', 'Invalid input')
}